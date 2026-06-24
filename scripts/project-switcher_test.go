package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSessionEntriesAndMatched(t *testing.T) {
	projects := map[string]string{
		"acme/alpha":   "/src/acme/alpha",
		"acme/bravo":   "/src/acme/bravo",
		"acme/charlie": "/src/acme/charlie",
	}
	sessions := []sessionInfo{
		{name: "acme/bravo"}, {name: "detached"}, {name: " feature"}, {name: "detached"}, {name: ""},
	}

	got, matched := sessionEntriesFor(projects, sessions)

	want := []entry{
		{kind: entryKindSession, name: sessionMarker + " feature", target: " feature"},
		{kind: entryKindSession, name: sessionMarker + "acme/bravo", path: "/src/acme/bravo", target: "acme/bravo"},
		{kind: entryKindSession, name: sessionMarker + "detached", target: "detached"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sessionEntriesFor() mismatch\nwant: %#v\ngot:  %#v", want, got)
	}

	wantMatched := map[string]struct{}{"acme/bravo": {}}
	if !reflect.DeepEqual(matched, wantMatched) {
		t.Fatalf("matched mismatch\nwant: %#v\ngot:  %#v", wantMatched, matched)
	}
}

// folderEntries lists only projects without a running session, so a project paired
// with a session ("acme/bravo") must be excluded.
func TestFolderEntriesExcludesMatched(t *testing.T) {
	projects := map[string]string{
		"acme/alpha":   "/src/acme/alpha",
		"acme/bravo":   "/src/acme/bravo",
		"acme/charlie": "/src/acme/charlie",
	}
	matched := map[string]struct{}{"acme/bravo": {}}

	got := folderEntries(projects, matched)

	want := []entry{
		{kind: entryKindFolder, name: "acme/alpha", path: "/src/acme/alpha", target: "acme/alpha"},
		{kind: entryKindFolder, name: "acme/charlie", path: "/src/acme/charlie", target: "acme/charlie"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("folderEntries() mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestFolderEntriesAllWhenNoSessions(t *testing.T) {
	projects := map[string]string{
		"acme/alpha": "/src/acme/alpha",
		"acme/bravo": "/src/acme/bravo",
	}

	got := folderEntries(projects, map[string]struct{}{})

	want := []entry{
		{kind: entryKindFolder, name: "acme/alpha", path: "/src/acme/alpha", target: "acme/alpha"},
		{kind: entryKindFolder, name: "acme/bravo", path: "/src/acme/bravo", target: "acme/bravo"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("folderEntries() mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}

// TestSessionEntriesWorktreeMatchedByPath covers the workmux case: the session
// name ("<prefix><handle>") does not match the worktree folder's project name, but
// its session_path does — so the two must be paired by path (folder suppressed via
// the matched set) and the entry must switch to the real session name.
func TestSessionEntriesWorktreeMatchedByPath(t *testing.T) {
	projects := map[string]string{
		"user/foo/main": "/src/host/user/foo/main",
		"user/bar":      "/src/host/user/bar",
	}
	sessions := []sessionInfo{
		{name: "wm-main-handle", path: "/src/host/user/foo/main"},
	}

	got, matched := sessionEntriesFor(projects, sessions)

	want := []entry{
		{kind: entryKindSession, name: sessionMarker + "user/foo/main", path: "/src/host/user/foo/main", target: "wm-main-handle"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sessionEntriesFor() mismatch\nwant: %#v\ngot:  %#v", want, got)
	}

	wantMatched := map[string]struct{}{"user/foo/main": {}}
	if !reflect.DeepEqual(matched, wantMatched) {
		t.Fatalf("matched mismatch\nwant: %#v\ngot:  %#v", wantMatched, matched)
	}

	// The matched worktree must not reappear as a folder.
	folders := folderEntries(projects, matched)
	wantFolders := []entry{
		{kind: entryKindFolder, name: "user/bar", path: "/src/host/user/bar", target: "user/bar"},
	}
	if !reflect.DeepEqual(folders, wantFolders) {
		t.Fatalf("folderEntries() mismatch\nwant: %#v\ngot:  %#v", wantFolders, folders)
	}
}

func TestCollectProjectsWorktreeAware(t *testing.T) {
	root := t.TempDir()

	mkdir := func(parts ...string) string {
		dir := filepath.Join(append([]string{root}, parts...)...)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %v: %v", parts, err)
		}
		return dir
	}
	gitDir := func(parts ...string) {
		if err := os.Mkdir(filepath.Join(append([]string{root}, append(parts, ".git")...)...), 0o755); err != nil {
			t.Fatalf("mkdir .git %v: %v", parts, err)
		}
	}
	gitFile := func(parts ...string) {
		if err := os.WriteFile(filepath.Join(append([]string{root}, append(parts, ".git")...)...), []byte("gitdir: elsewhere\n"), 0o644); err != nil {
			t.Fatalf("write .git %v: %v", parts, err)
		}
	}

	// Plain repo: .git directory at project depth.
	plain := mkdir("host", "user", "plainrepo")
	gitDir("host", "user", "plainrepo")

	// Worktree container: no .git of its own, holds worktree children plus a stray dir.
	mkdir("host", "user", "foo", "main")
	gitDir("host", "user", "foo", "main") // main worktree -> .git directory
	mkdir("host", "user", "foo", "feat")
	gitFile("host", "user", "foo", "feat") // linked worktree -> .git file
	mkdir("host", "user", "foo", "notes")  // not a worktree -> skipped
	mainWT := filepath.Join(root, "host", "user", "foo", "main")
	featWT := filepath.Join(root, "host", "user", "foo", "feat")

	// Plain non-git folder with a non-git child: legacy fallback, recorded as-is.
	plainFolder := mkdir("host", "user", "plainfolder", "sub")
	plainFolder = filepath.Dir(plainFolder)

	got, err := collectProjects(config{root: root, projectDepth: 3, nameDepth: 2})
	if err != nil {
		t.Fatalf("collectProjects: %v", err)
	}

	want := map[string]string{
		"user/plainrepo":   plain,
		"user/foo/main":    mainWT,
		"user/foo/feat":    featWT,
		"user/plainfolder": plainFolder,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collectProjects() mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestSessionEntriesOnly(t *testing.T) {
	sessions := []sessionInfo{
		{name: "detached"}, {name: "alpha"}, {name: " feature"}, {name: " bugfix"}, {name: "detached"},
	}

	got, matched := sessionEntriesFor(nil, sessions)

	want := []entry{
		{kind: entryKindSession, name: sessionMarker + " bugfix", target: " bugfix"},
		{kind: entryKindSession, name: sessionMarker + " feature", target: " feature"},
		{kind: entryKindSession, name: sessionMarker + "alpha", target: "alpha"},
		{kind: entryKindSession, name: sessionMarker + "detached", target: "detached"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sessionEntriesFor() mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
	if len(matched) != 0 {
		t.Fatalf("matched should be empty, got: %#v", matched)
	}
}
