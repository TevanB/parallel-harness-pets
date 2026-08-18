package visits

import (
	"testing"
	"time"
)

// A den has to remember everyone who worked in it, including agents that have
// since moved on. That is the whole point: the live register forgets them.
func TestDenRemembersEveryoneWhoPassedThrough(t *testing.T) {
	dir := t.TempDir()
	start := time.Now()
	for _, session := range []string{"first", "second"} {
		if err := Record(dir, "demo#alpha", session, start); err != nil {
			t.Fatal(err)
		}
	}
	// The same agent moving to another den must not be forgotten by the one it
	// left, which is exactly what the live register does.
	if err := Record(dir, "demo#beta", "first", start); err != nil {
		t.Fatal(err)
	}
	if seen := ForDen(dir, "demo#alpha"); len(seen) != 2 {
		t.Errorf("alpha remembers %d visitors, want 2", len(seen))
	}
	if seen := ForDen(dir, "demo#beta"); len(seen) != 1 {
		t.Errorf("beta remembers %d visitors, want 1", len(seen))
	}
	if seen := ForDen(dir, "demo#never-visited"); len(seen) != 0 {
		t.Errorf("an unvisited den remembers %d", len(seen))
	}
}

// First arrival is the fact worth keeping. Rewriting it on every render would
// turn the history into a clock.
func TestFirstArrivalSurvivesLaterVisits(t *testing.T) {
	dir := t.TempDir()
	arrived := time.Now()
	if err := Record(dir, "demo#alpha", "wanderer", arrived); err != nil {
		t.Fatal(err)
	}
	returned := arrived.Add(4 * time.Hour)
	if err := Record(dir, "demo#alpha", "wanderer", returned); err != nil {
		t.Fatal(err)
	}
	seen := ForDen(dir, "demo#alpha")
	if len(seen) != 1 {
		t.Fatalf("one agent produced %d entries", len(seen))
	}
	if !seen[0].First.Equal(arrived.Truncate(time.Second)) {
		t.Errorf("first arrival moved: %v, want %v", seen[0].First, arrived)
	}
	if !seen[0].Last.After(seen[0].First) {
		t.Error("last seen did not advance on a return visit")
	}
}

// The collection needs to ask which places have been worked in, which means
// reading distinct den keys back out of a directory of per-session files.
func TestDensListsEachDenOnce(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	for _, v := range [][2]string{
		{"demo#alpha", "one"}, {"demo#alpha", "two"}, {"demo#alpha", "three"},
		{"demo#beta", "one"}, {"other#alpha", "one"},
	} {
		if err := Record(dir, v[0], v[1], now); err != nil {
			t.Fatal(err)
		}
	}
	dens := Dens(dir)
	if len(dens) != 3 {
		t.Fatalf("got %d dens %v, want 3 distinct", len(dens), dens)
	}
	// Sorted, so the caller gets a stable order between runs.
	if dens[0] != "demo#alpha" || dens[1] != "demo#beta" || dens[2] != "other#alpha" {
		t.Errorf("unsorted or wrong: %v", dens)
	}
	if empty := Dens(t.TempDir()); len(empty) != 0 {
		t.Errorf("a fresh state dir reported %d dens", len(empty))
	}
}
