package score

import (
	"testing"

	"github.com/TevanB/parallel-harness-pets/internal/config"
	"github.com/TevanB/parallel-harness-pets/internal/state"
)

func TestScoringMatchesTheShellRules(t *testing.T) {
	settings := config.Default()
	cases := []struct {
		name   string
		state  state.State
		tests  string
		hearts int
	}{
		{"clean", state.State{}, "unknown", 5},
		{"one dirty file", state.State{Dirty: 1}, "unknown", 4},
		{"past the dirty threshold", state.State{Dirty: 16}, "unknown", 3},
		{"one unpushed commit", state.State{Unpushed: 1}, "unknown", 4},
		{"past the unpushed threshold", state.State{Unpushed: 6}, "unknown", 3},
		{"failing tests", state.State{}, "fail", 3},
		{"passing tests cost nothing", state.State{}, "pass", 5},
		{"behind never costs hearts", state.State{Behind: 500}, "unknown", 5},
		{"everything at once floors at zero", state.State{Dirty: 40, Unpushed: 40}, "fail", 0},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := Of(testCase.state, testCase.tests, settings).Hearts; got != testCase.hearts {
				t.Errorf("hearts = %d, want %d", got, testCase.hearts)
			}
		})
	}
}

// The migration penalty was the heaviest in the model and applied to everyone.
// It is now opt-in, so a repo with no migrations is never charged for them.
func TestMigrationPenaltyIsOffByDefault(t *testing.T) {
	split := state.State{Migrations: 3}
	if got := Of(split, "unknown", config.Default()).Hearts; got != 5 {
		t.Errorf("default config charged %d hearts for migrations, want 0 penalty", 5-got)
	}

	optedIn := config.Default()
	optedIn.Signals.Migrations.Enabled = true
	if got := Of(split, "unknown", optedIn).Hearts; got != 3 {
		t.Errorf("opted-in config gave %d hearts, want 3", got)
	}
}

// Scoring is data now, so a user can tune or disable any signal.
func TestDisablingASignalStopsItsPenalty(t *testing.T) {
	settings := config.Default()
	settings.Signals.Enabled = []string{"unpushed", "behind"}
	if got := Of(state.State{Dirty: 99}, "fail", settings).Hearts; got != 5 {
		t.Errorf("hearts = %d with dirty and tests disabled, want 5", got)
	}
}

func TestPenaltiesExplainThemselves(t *testing.T) {
	result := Of(state.State{Dirty: 20, Unpushed: 1}, "fail", config.Default())
	seen := map[string]int{}
	for _, penalty := range result.Penalties {
		seen[penalty.Signal] += penalty.Cost
	}
	if seen["uncommitted"] != 2 || seen["unpushed"] != 1 || seen["tests"] != 2 {
		t.Errorf("penalties = %v, want uncommitted:2 unpushed:1 tests:2", seen)
	}
}

func TestFaceDegradesWithHearts(t *testing.T) {
	previous := ""
	for hearts := 5; hearts >= 0; hearts-- {
		face := Face(hearts)
		if face == "" || face == previous {
			t.Errorf("Face(%d) = %q, want a distinct non-empty face", hearts, face)
		}
		previous = face
	}
}
