// Package score turns hygiene counts into hearts and a face.
package score

import (
	"github.com/TevanB/parallel-harness-pets/internal/config"
	"github.com/TevanB/parallel-harness-pets/internal/state"
)

const Max = 5

// Penalty is one reason the pet lost hearts, so the card can explain itself.
type Penalty struct {
	Signal string
	Cost   int
}

type Result struct {
	Hearts    int
	Penalties []Penalty
}

func Of(current state.State, tests string, settings config.Config) Result {
	scoring := settings.Score
	result := Result{Hearts: scoring.Start}

	charge := func(signal string, cost int, when bool) {
		if !when || cost <= 0 {
			return
		}
		result.Hearts -= cost
		result.Penalties = append(result.Penalties, Penalty{Signal: signal, Cost: cost})
	}

	if settings.SignalEnabled("dirty") {
		charge("uncommitted", scoring.DirtyAny, current.Dirty > 0)
		charge("uncommitted", scoring.DirtyMany, current.Dirty > scoring.DirtyManyAt)
	}
	if settings.SignalEnabled("unpushed") {
		charge("unpushed", scoring.UnpushedAny, current.Unpushed > 0)
		charge("unpushed", scoring.UnpushedMany, current.Unpushed > scoring.UnpushedManyAt)
	}
	if settings.SignalEnabled("tests") {
		charge("tests", scoring.TestsFailing, tests == "fail")
	}
	if settings.Signals.Migrations.Enabled {
		charge("migrations", settings.Signals.Migrations.Penalty, current.Migrations > 1)
	}

	if result.Hearts < 0 {
		result.Hearts = 0
	}
	if result.Hearts > Max {
		result.Hearts = Max
	}
	return result
}

func Face(hearts int) string {
	switch {
	case hearts >= 5:
		return "•ᴗ•"
	case hearts == 4:
		return "•_•"
	case hearts == 3:
		return "¬_¬"
	case hearts == 2:
		return ">_<"
	case hearts == 1:
		return "@_@"
	default:
		return "x_x"
	}
}
