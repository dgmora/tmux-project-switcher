package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMergeEntriesMixedSections(t *testing.T) {
	projects := map[string]string{
		"acme/alpha":   "/src/acme/alpha",
		"acme/bravo":   "/src/acme/bravo",
		"acme/charlie": "/src/acme/charlie",
	}
	sessions := []string{"acme/bravo", "detached", " feature", "detached", ""}

	got := mergeEntries(projects, sessions)

	want := []entry{
		{kind: entryKindSession, name: " feature"},
		{kind: entryKindSession, name: "acme/bravo", path: "/src/acme/bravo"},
		{kind: entryKindSession, name: "detached"},
		{kind: entryKindDivider, name: dividerName},
		{kind: entryKindFolder, name: "acme/alpha", path: "/src/acme/alpha"},
		{kind: entryKindFolder, name: "acme/charlie", path: "/src/acme/charlie"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mergeEntries() mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestMergeEntriesWithoutSessionsOmitsDivider(t *testing.T) {
	projects := map[string]string{
		"acme/alpha": "/src/acme/alpha",
		"acme/bravo": "/src/acme/bravo",
	}

	got := mergeEntries(projects, nil)

	want := []entry{
		{kind: entryKindFolder, name: "acme/alpha", path: "/src/acme/alpha"},
		{kind: entryKindFolder, name: "acme/bravo", path: "/src/acme/bravo"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mergeEntries() mismatch\nwant: %#v\ngot:  %#v", want, got)
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

func TestMergeEntriesOnlySessionsOmitsDivider(t *testing.T) {
	sessions := []string{"detached", "alpha", " feature", " bugfix", "detached"}

	got := mergeEntries(nil, sessions)

	want := []entry{
		{kind: entryKindSession, name: " bugfix"},
		{kind: entryKindSession, name: " feature"},
		{kind: entryKindSession, name: "alpha"},
		{kind: entryKindSession, name: "detached"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mergeEntries() mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}
