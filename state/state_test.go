package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreLoadSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	s := NewStore(path)
	s.UpdateState(func(st *State) {
		st.Chores["c1"] = "completed"
		st.Rewards["r1"] = true
		st.AllCompletedFired["2026-03-25:cat1"] = true
		st.LastPollAt = time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC)
	})

	// Load into a new store.
	s2 := NewStore(path)
	if err := s2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	st := s2.GetState()
	if st.Chores["c1"] != "completed" {
		t.Errorf("chore c1 = %q, want completed", st.Chores["c1"])
	}
	if !st.Rewards["r1"] {
		t.Error("reward r1 should be true")
	}
	if !st.AllCompletedFired["2026-03-25:cat1"] {
		t.Error("all_completed_fired should be set")
	}
}

func TestStoreLoadNonExistent(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "missing.json"))
	if err := s.Load(); err != nil {
		t.Fatalf("Load should return nil for missing file, got: %v", err)
	}
}

func TestStoreLoadCorrupt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	os.WriteFile(path, []byte("{invalid json"), 0o600)

	s := NewStore(path)
	if err := s.Load(); err == nil {
		t.Fatal("expected error for corrupt file")
	}
}

func TestStoreAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "state.json")

	s := NewStore(path)
	s.UpdateState(func(st *State) {
		st.Chores["c1"] = "pending"
	})

	// Verify the temp file was cleaned up and final file exists.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("state file should exist: %v", err)
	}
	tmpPath := path + ".tmp"
	if _, err := os.Stat(tmpPath); err == nil {
		t.Error("temp file should not exist after rename")
	}
}

func TestGetStateCopy(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "state.json"))
	s.UpdateState(func(st *State) {
		st.Chores["c1"] = "pending"
	})

	// Mutating the returned copy should not affect the store.
	st := s.GetState()
	st.Chores["c1"] = "modified"

	st2 := s.GetState()
	if st2.Chores["c1"] != "pending" {
		t.Errorf("store was mutated via returned copy")
	}
}
