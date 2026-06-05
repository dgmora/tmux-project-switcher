package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type config struct {
	root         string
	projectDepth int
	nameDepth    int
}

type entryKind string

const (
	entryKindFolder  entryKind = "folder"
	entryKindSession entryKind = "session"
	entryKindDivider entryKind = "divider"

	dividerName         = "────────────────────────────────────────────────"
	sessionBranchPrefix = ""
)

// sessionMarker is prepended to the display text of session rows. fzf drops the
// section divider while you type, so this marker is what distinguishes a live
// session from a plain folder once both are filtered into the same view. The glyph
// is workmux's own native session prefix (nf-oct-git_branch + space), so our
// de-duplicated rows render identically to real workmux session rows.
const sessionMarker = " "

type entry struct {
	kind   entryKind
	name   string // display text (fzf column)
	path   string // project dir, used for `tmux new-session -c`
	target string // tmux session name: has-session / switch-client / new-session -s
}

type sessionInfo struct {
	name string
	path string
}

func main() {
	cfg := config{
		root:         defaultRoot(),
		projectDepth: envInt("TMUX_PROJECT_SWITCHER_PROJECT_DEPTH", 3),
		nameDepth:    envInt("TMUX_PROJECT_SWITCHER_FOLDERS_AMOUNT", 2),
	}

	rootFlag := flag.String("root", cfg.root, "Root directory containing projects")
	depthFlag := flag.Int("project-depth", cfg.projectDepth, "Directory depth to treat as projects")
	nameDepthFlag := flag.Int("name-depth", cfg.nameDepth, "How many trailing path segments make the project name")
	flag.Parse()

	cfg.root = *rootFlag
	cfg.projectDepth = *depthFlag
	cfg.nameDepth = *nameDepthFlag

	if cfg.projectDepth < 1 {
		exitWithError(errors.New("project-depth must be >= 1"))
	}
	if cfg.nameDepth < 1 {
		exitWithError(errors.New("name-depth must be >= 1"))
	}

	projectCh := make(chan result[map[string]string], 1)
	sessionCh := make(chan result[[]sessionInfo], 1)

	go func() {
		data, err := collectProjects(cfg)
		projectCh <- result[map[string]string]{data: data, err: err}
	}()

	go func() {
		data, err := listSessions()
		sessionCh <- result[[]sessionInfo]{data: data, err: err}
	}()

	projectRes := <-projectCh
	if projectRes.err != nil {
		exitWithError(projectRes.err)
	}

	sessionRes := <-sessionCh
	if sessionRes.err != nil {
		exitWithError(sessionRes.err)
	}

	entries := mergeEntries(projectRes.data, sessionRes.data)
	for _, e := range entries {
		fmt.Printf("%s\t%s\t%s\t%s\n", e.kind, e.name, e.path, e.target)
	}
}

type result[T any] struct {
	data T
	err  error
}

func mergeEntries(projects map[string]string, sessions []sessionInfo) []entry {
	// Index projects by directory so a session can be paired with the folder it was
	// opened from. workmux roots each worktree session at the worktree dir, so the
	// session's #{session_path} equals the project path even though their names differ
	// (folder: "$user/$repo/$worktree"; session: "<prefix><handle>").
	byPath := make(map[string]string, len(projects))
	for name, path := range projects {
		byPath[filepath.Clean(path)] = name
	}

	matched := make(map[string]struct{}, len(sessions)) // project names represented by a session
	seenSessions := make(map[string]struct{}, len(sessions))
	sessionEntries := make([]entry, 0, len(sessions))

	for _, sess := range sessions {
		if sess.name == "" {
			continue
		}
		if _, dup := seenSessions[sess.name]; dup {
			continue
		}
		seenSessions[sess.name] = struct{}{}

		// Resolve the project this session represents: by path first (handles
		// worktrees, whose names never match), then by name (plain repos / sessions
		// created by this switcher).
		projName, projPath := "", ""
		if sess.path != "" {
			if n, ok := byPath[filepath.Clean(sess.path)]; ok {
				projName, projPath = n, projects[n]
			}
		}
		if projName == "" {
			if p, ok := projects[sess.name]; ok {
				projName, projPath = sess.name, p
			}
		}

		if projName != "" {
			matched[projName] = struct{}{}
			// Display the project path (with context), but switch to the real session.
			sessionEntries = append(sessionEntries, entry{
				kind:   entryKindSession,
				name:   sessionMarker + projName,
				path:   projPath,
				target: sess.name,
			})
			continue
		}
		sessionEntries = append(sessionEntries, entry{
			kind:   entryKindSession,
			name:   sessionMarker + sess.name,
			target: sess.name,
		})
	}

	folders := make([]entry, 0, len(projects))
	for name, path := range projects {
		if _, ok := matched[name]; ok {
			continue
		}
		folders = append(folders, entry{kind: entryKindFolder, name: name, path: path, target: name})
	}

	sortEntriesByName(folders)
	sortSessionEntriesForDefaultLayout(sessionEntries)

	entries := make([]entry, 0, len(folders)+len(sessionEntries)+1)
	// fzf's default layout renders earlier input lines lower in the popup.
	// Emit entries in reverse visual order so the picker shows folders above
	// the divider and sessions below it.
	entries = append(entries, sessionEntries...)
	if len(folders) > 0 && len(sessionEntries) > 0 {
		entries = append(entries, entry{kind: entryKindDivider, name: dividerName})
	}
	entries = append(entries, folders...)

	return entries
}

func collectProjects(cfg config) (map[string]string, error) {
	root := filepath.Clean(cfg.root)

	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat root %q: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root %q is not a directory", root)
	}

	projects := make(map[string]string)

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, fs.ErrPermission) {
				return filepath.SkipDir
			}
			return walkErr
		}

		if !d.IsDir() {
			return nil
		}

		if path == root {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		depth := segmentCount(rel)

		if depth > cfg.projectDepth {
			return filepath.SkipDir
		}

		if depth == cfg.projectDepth {
			recordProject(projects, path, cfg)
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return projects, nil
}

func listSessions() ([]sessionInfo, error) {
	// session_path is the session's start directory; unlike pane_current_path it is
	// stable and equals the worktree dir workmux passed to `new-session -c`.
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}\t#{session_path}")
	output, err := cmd.Output()
	if err != nil {
		var execErr *exec.ExitError
		if errors.As(err, &execErr) {
			// No tmux server running; treat as zero sessions.
			return []sessionInfo{}, nil
		}
		return nil, fmt.Errorf("tmux list-sessions: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	sessions := make([]sessionInfo, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		name, path, _ := strings.Cut(line, "\t")
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		sessions = append(sessions, sessionInfo{name: name, path: strings.TrimSpace(path)})
	}

	return sessions, nil
}

func sortEntriesByName(entries []entry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})
}

func sortSessionEntriesForDefaultLayout(entries []entry) {
	sort.Slice(entries, func(i, j int) bool {
		iIsBranch := strings.HasPrefix(entries[i].name, sessionBranchPrefix)
		jIsBranch := strings.HasPrefix(entries[j].name, sessionBranchPrefix)
		if iIsBranch != jIsBranch {
			// Branch sessions are emitted earlier so fzf renders them lower than
			// regular sessions in the default layout.
			return iIsBranch
		}
		return entries[i].name < entries[j].name
	})
}

// recordProject classifies a directory found at the configured project depth.
// A plain working tree (a .git entry exists — directory or file) is recorded as-is.
// A worktree container (no .git, but holds child worktrees) is expanded: each child
// worktree is recorded one level deeper, with the child appended to the project name.
// Anything else keeps the legacy behavior and is recorded as-is.
func recordProject(projects map[string]string, path string, cfg config) {
	if isWorktree(path) {
		projects[projectNameFromPath(path, cfg.nameDepth)] = path
		return
	}
	children := worktreeChildren(path)
	if len(children) == 0 {
		projects[projectNameFromPath(path, cfg.nameDepth)] = path
		return
	}
	for _, child := range children {
		projects[projectNameFromPath(child, cfg.nameDepth+1)] = child
	}
}

// isWorktree reports whether dir contains a .git entry (directory for a plain repo or
// main worktree, file for a linked worktree).
func isWorktree(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

// worktreeChildren returns the immediate subdirectories of dir that are themselves worktrees.
func worktreeChildren(dir string) []string {
	dirents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var children []string
	for _, de := range dirents {
		if !de.IsDir() {
			continue
		}
		child := filepath.Join(dir, de.Name())
		if isWorktree(child) {
			children = append(children, child)
		}
	}
	return children
}

func projectNameFromPath(path string, depth int) string {
	cleaned := filepath.ToSlash(filepath.Clean(path))
	segments := strings.Split(cleaned, "/")
	if depth > len(segments) {
		depth = len(segments)
	}
	start := len(segments) - depth
	if start < 0 {
		start = 0
	}
	return strings.Join(segments[start:], "/")
}

func segmentCount(rel string) int {
	if rel == "." || rel == "" {
		return 0
	}
	return strings.Count(filepath.ToSlash(rel), "/") + 1
}

func envInt(key string, def int) int {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	var parsed int
	_, err := fmt.Sscanf(val, "%d", &parsed)
	if err != nil {
		return def
	}
	if parsed < 1 {
		return def
	}
	return parsed
}

func defaultRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, "src")
}

func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}
