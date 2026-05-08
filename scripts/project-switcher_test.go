package main

import (
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
