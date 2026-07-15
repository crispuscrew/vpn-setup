package main

import (
	"errors"
	"testing"
)

// planImports must skip already-tracked users, mint a link for the rest, surface
// mint failures separately, and return both lists sorted for a stable reply.
func TestPlanImports(t *testing.T) {
	tracked := map[string]bool{"claimed": true}
	tokenFor := func(name string) (string, error) {
		if name == "broken" {
			return "", errors.New("mint failed")
		}
		return "tok-" + name, nil
	}

	done, failed := planImports(
		[]string{"charlie", "claimed", "alice", "broken"},
		func(name string) bool { return tracked[name] },
		tokenFor,
		"vpnbot",
	)

	wantDone := []importResult{
		{Username: "alice", Link: "https://t.me/vpnbot?start=tok-alice"},
		{Username: "charlie", Link: "https://t.me/vpnbot?start=tok-charlie"},
	}
	if len(done) != len(wantDone) {
		t.Fatalf("done = %v, want %v", done, wantDone)
	}
	for i, res := range done {
		if res != wantDone[i] {
			t.Errorf("done[%d] = %+v, want %+v", i, res, wantDone[i])
		}
	}
	if len(failed) != 1 || failed[0] != "broken" {
		t.Errorf("failed = %v, want [broken]", failed)
	}
}

// A roster whose users are all already tracked yields nothing to import.
func TestPlanImportsAllTracked(t *testing.T) {
	done, failed := planImports(
		[]string{"a", "b"},
		func(string) bool { return true },
		func(name string) (string, error) { return "tok", nil },
		"vpnbot",
	)
	if len(done) != 0 || len(failed) != 0 {
		t.Errorf("want empty, got done=%v failed=%v", done, failed)
	}
}
