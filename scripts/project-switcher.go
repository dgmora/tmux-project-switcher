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

type entry struct {
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
    sessionCh := make(chan result[[]string], 1)

    go func() {
        data, err := collectProjects(cfg)
        projectCh <- result[map[string]string]{data: data, err: err}
    }()

    go func() {
        data, err := listSessions()
        sessionCh <- result[[]string]{data: data, err: err}
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
        fmt.Printf("%s\t%s\n", e.name, e.path)
    }
}

type result[T any] struct {
    data T
    err  error
}

func mergeEntries(projects map[string]string, sessions []string) []entry {
    entries := make([]entry, 0, len(projects)+len(sessions))
    seen := make(map[string]struct{}, len(projects)+len(sessions))

    for name, path := range projects {
        entries = append(entries, entry{name: name, path: path})
        seen[name] = struct{}{}
    }

    for _, sess := range sessions {
        if sess == "" {
            continue
        }
        if _, ok := seen[sess]; ok {
            continue
        }
        entries = append(entries, entry{name: sess})
        seen[sess] = struct{}{}
    }

    sort.Slice(entries, func(i, j int) bool {
        return entries[i].name < entries[j].name
    })

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
            name := projectNameFromPath(path, cfg.nameDepth)
            projects[name] = path
            return filepath.SkipDir
        }

        return nil
    })

    if err != nil {
        return nil, err
    }

    return projects, nil
}

func listSessions() ([]string, error) {
    cmd := exec.Command("tmux", "list-sessions", "-F", "#S")
    output, err := cmd.Output()
    if err != nil {
        var execErr *exec.ExitError
        if errors.As(err, &execErr) {
            // No tmux server running; treat as zero sessions.
            return []string{}, nil
        }
        return nil, fmt.Errorf("tmux list-sessions: %w", err)
    }

    lines := strings.Split(string(output), "\n")
    sessions := make([]string, 0, len(lines))
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        if trimmed != "" {
            sessions = append(sessions, trimmed)
        }
    }

    return sessions, nil
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
