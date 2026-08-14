package agents

import (
	"path/filepath"
	"testing"
	"time"
)

// The whole reason this package exists: two agents in one worktree used to
// overwrite each other's state, so a den could only ever show one of them.
func TestTwoAgentsShareADen(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	for _, session := range []string{"aaaa-1111", "bbbb-2222"} {
		if err := Touch(dir, Record{Session: session, Den: "demo#feat", Root: dir}, now); err != nil {
			t.Fatal(err)
		}
	}
	if living := InDen(dir, "demo#feat", now); len(living) != 2 {
		t.Fatalf("den holds %d agents, want 2", len(living))
	}
	if other := InDen(dir, "demo#elsewhere", now); len(other) != 0 {
		t.Errorf("agents leaked into another den: %d", len(other))
	}
}

// An agent that exits never says so, so silence has to be what marks it stale,
// and a long enough silence has to remove it entirely.
func TestSilenceGoesStaleThenForgotten(t *testing.T) {
	dir := t.TempDir()
	start := time.Now()
	if err := Touch(dir, Record{Session: "aaaa-1111", Den: "demo#feat", Root: dir}, start); err != nil {
		t.Fatal(err)
	}
	if All(dir, start)[0].Stale(start) {
		t.Error("a just-seen agent is stale")
	}
	later := start.Add(StaleAfter + time.Second)
	if !All(dir, later)[0].Stale(later) {
		t.Error("a silent agent never went stale")
	}
	if living := All(dir, start.Add(Forgotten+time.Minute)); len(living) != 0 {
		t.Errorf("a long-silent agent was still listed: %d", len(living))
	}
}

// An agent can move between worktrees. Its identity travels with it, it leaves
// no ghost behind, and the arrival time resets so travel is visible.
func TestAgentJumpsWorktrees(t *testing.T) {
	dir := t.TempDir()
	start := time.Now()
	if err := Touch(dir, Record{Session: "aaaa-1111", Den: "demo#first", Root: dir}, start); err != nil {
		t.Fatal(err)
	}
	// Well inside the heartbeat window: a move must still be written at once,
	// or the arrival is lost and the agent appears in the wrong den.
	moved := start.Add(2 * time.Second)
	if err := Touch(dir, Record{Session: "aaaa-1111", Den: "demo#second", Root: dir}, moved); err != nil {
		t.Fatal(err)
	}
	if left := InDen(dir, "demo#first", moved); len(left) != 0 {
		t.Errorf("a ghost stayed behind in the old den: %d", len(left))
	}
	arrived := InDen(dir, "demo#second", moved)
	if len(arrived) != 1 {
		t.Fatalf("agent did not arrive in the new den: %d", len(arrived))
	}
	if !arrived[0].JustArrived(moved) {
		t.Error("arrival time did not reset on the move")
	}
	// Staying put must not keep resetting the arrival clock.
	settled := moved.Add(Heartbeat + time.Second)
	if err := Touch(dir, Record{Session: "aaaa-1111", Den: "demo#second", Root: dir}, settled); err != nil {
		t.Fatal(err)
	}
	if InDen(dir, "demo#second", settled)[0].Since != arrived[0].Since {
		t.Error("a heartbeat in the same den reset the arrival time")
	}
}

// An agent outside any worktree, or one whose worktree was deleted underneath
// it, is still running. Neither may retire it: only silence does.
func TestLocationDoesNotDecideExistence(t *testing.T) {
	dir := t.TempDir()
	start := time.Now()
	if err := Touch(dir, Record{Session: "aaaa-1111", Den: "demo#gone",
		Root: filepath.Join(dir, "deleted-worktree")}, start); err != nil {
		t.Fatal(err)
	}
	if living := All(dir, start); len(living) != 1 {
		t.Fatalf("a live agent was pruned for a missing worktree: %d", len(living))
	}
	// Beat keeps it alive without knowing where it is, holding the last den.
	later := start.Add(Heartbeat + time.Second)
	if err := Beat(dir, "aaaa-1111", later); err != nil {
		t.Fatal(err)
	}
	held := All(dir, later)
	if len(held) != 1 || held[0].Den != "demo#gone" {
		t.Fatalf("Beat lost the agent or its den: %+v", held)
	}
	if held[0].Stale(later) {
		t.Error("Beat did not refresh the agent's liveness")
	}
}

// Label is what a human reads in a list, so it must never come back blank.
func TestLabelFallsBackToTheSessionID(t *testing.T) {
	named := Record{Session: "aaaa-1111-bbbb", Name: "fix the flaky auth test"}
	if named.Label() != "fix the flaky auth test" {
		t.Errorf("named session labelled %q", named.Label())
	}
	if unnamed := (Record{Session: "aaaa-1111-bbbb"}).Label(); unnamed != "aaaa-111" {
		t.Errorf("unnamed session labelled %q", unnamed)
	}
}
