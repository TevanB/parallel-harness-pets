package identity

import "testing"

// The roster is the contribution surface, so the rules that make a city
// acceptable have to be enforced by the suite rather than by review.
func TestPlaceRosterIsWellFormed(t *testing.T) {
	if err := ValidatePlaces(); err != nil {
		t.Fatal(err)
	}
}

// A den is a location. Changing branch inside a worktree must not move it, and
// the same repo and worktree must land on the same city on any machine.
func TestPlaceIsStableAndRepoScoped(t *testing.T) {
	first := PlaceFor("demo", "feat-checkout")
	if again := PlaceFor("demo", "feat-checkout"); again.Code != first.Code {
		t.Errorf("same repo and worktree gave %s then %s", first.Code, again.Code)
	}
	// Every repository having its own main checkout land on one city was the
	// bug this keying exists to avoid.
	if other := PlaceFor("other", "feat-checkout"); other.Code == first.Code {
		t.Errorf("different repos both resolved to %s", first.Code)
	}
}
