// Package render turns a resolved pet into text for whichever surface asked.
//
// One code path feeds the Claude Code status line, a tmux segment, a terminal
// title, and JSON for editor extensions.
package render

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/TevanB/parallel-harness-pets/internal/config"
	"github.com/TevanB/parallel-harness-pets/internal/identity"
	"github.com/TevanB/parallel-harness-pets/internal/score"
	"github.com/TevanB/parallel-harness-pets/internal/state"
)

const (
	reset    = "\033[0m"
	dim      = "\033[38;5;240m"
	label    = "\033[38;5;252m"
	warn     = "\033[38;5;179m"
	heartHue = "\033[38;5;210m"
)

// View is everything a surface needs, already resolved.
type View struct {
	Pet      identity.Pet
	Branch   string
	Root     string
	Score    score.Result
	State    state.State
	Tests    string
	Model    string
	HasState bool
}

func hue(color int) string {
	return fmt.Sprintf("\033[38;5;%dm", color)
}

func (v View) face() string {
	if !v.HasState {
		return "-.-"
	}
	return score.Face(v.Score.Hearts)
}

// Body is the creature itself, frame and all.
func (v View) Body() string {
	return v.Pet.Prefix + v.face() + v.Pet.Suffix
}

func (v View) hearts() string {
	if !v.HasState {
		return dim + "·····" + reset
	}
	filled := strings.Repeat("♥", v.Score.Hearts)
	empty := strings.Repeat("♡", score.Max-v.Score.Hearts)
	return heartHue + filled + dim + empty + reset
}

// Flags are the raw counts behind the mood, shown only when they are non-zero.
func (v View) Flags(settings config.Config) []string {
	var flags []string
	if v.State.Dirty > 0 {
		flags = append(flags, fmt.Sprintf("%d△", v.State.Dirty))
	}
	if v.State.Unpushed > 0 {
		flags = append(flags, fmt.Sprintf("%d↑", v.State.Unpushed))
	}
	if v.State.Behind > settings.Display.BehindShownPast {
		flags = append(flags, fmt.Sprintf("%d↓", v.State.Behind))
	}
	if v.State.Migrations > 1 {
		flags = append(flags, fmt.Sprintf("⑂%d", v.State.Migrations))
	}
	if v.Tests == "fail" {
		flags = append(flags, "✗")
	}
	return flags
}

func truncate(text string, max int) string {
	if max <= 1 || len([]rune(text)) <= max {
		return text
	}
	return string([]rune(text)[:max-1]) + "…"
}

// Statusline is the one-line form Claude Code refreshes every second.
func Statusline(v View, settings config.Config) string {
	color := hue(v.Pet.Color)
	parts := []string{
		color + v.Body() + reset,
		color + v.Pet.Name + reset,
		dim + "·" + reset,
		label + truncate(v.Branch, settings.Display.BranchLabelMax) + reset,
		v.hearts(),
	}
	if flags := v.Flags(settings); v.HasState && len(flags) > 0 {
		parts = append(parts, warn+strings.Join(flags, " ")+reset)
	}
	if v.Model != "" {
		parts = append(parts, dim+"·"+reset, dim+v.Model+reset)
	}
	return strings.Join(parts, " ")
}

// Tmux uses tmux's own colour syntax so the segment inherits the bar's styling.
func Tmux(v View, settings config.Config) string {
	out := fmt.Sprintf("#[fg=colour%d]%s %s#[default] %s",
		v.Pet.Color, v.Body(), v.Pet.Name, truncate(v.Branch, settings.Display.BranchLabelMax))
	if flags := v.Flags(settings); v.HasState && len(flags) > 0 {
		out += " #[fg=colour179]" + strings.Join(flags, " ") + "#[default]"
	}
	return out
}

// Title is plain text for a terminal or tab title, where escapes cannot go.
func Title(v View, settings config.Config) string {
	out := fmt.Sprintf("%s %s · %s", v.Body(), v.Pet.Name,
		truncate(v.Branch, settings.Display.BranchLabelMax))
	if flags := v.Flags(settings); v.HasState && len(flags) > 0 {
		out += " " + strings.Join(flags, " ")
	}
	return out
}

// JSON is the contract for editor extensions and any surface not yet invented.
func JSON(v View, settings config.Config) (string, error) {
	payload := map[string]any{
		"species":  v.Pet.Name,
		"body":     v.Body(),
		"color":    v.Pet.Color,
		"branch":   v.Branch,
		"root":     v.Root,
		"hearts":   v.Score.Hearts,
		"max":      score.Max,
		"flags":    v.Flags(settings),
		"tests":    v.Tests,
		"ready":    v.HasState,
		"dirty":    v.State.Dirty,
		"unpushed": v.State.Unpushed,
		"behind":   v.State.Behind,
	}
	if len(v.State.External) > 0 {
		payload["external"] = v.State.External
	}
	encoded, err := json.Marshal(payload)
	return string(encoded), err
}

// Card is the full readout, with every penalised signal called out in amber.
func Card(v View, settings config.Config) string {
	var out strings.Builder
	color := hue(v.Pet.Color)

	penalised := map[string]bool{}
	for _, penalty := range v.Score.Penalties {
		penalised[penalty.Signal] = true
	}
	row := func(name, value string, flagged bool) {
		shown := value
		if flagged {
			shown = warn + value + reset
		}
		fmt.Fprintf(&out, "  %s%-14s%s %s\n", dim, name, reset, shown)
	}

	fmt.Fprintf(&out, "\n  %s%s%s  %s%s%s\n", color, v.Body(), reset, color, v.Pet.Name, reset)
	fmt.Fprintf(&out, "  %s%s%s\n", dim, strings.Repeat("─", 30), reset)
	row("branch", v.Branch, false)
	row("worktree", v.Root, false)
	row("mood", fmt.Sprintf("%s  %d/%d", v.hearts(), v.Score.Hearts, score.Max), false)
	fmt.Fprintln(&out)
	row("uncommitted", fmt.Sprint(v.State.Dirty), penalised["uncommitted"])
	row("unpushed", fmt.Sprint(v.State.Unpushed), penalised["unpushed"])
	row("behind trunk", fmt.Sprint(v.State.Behind), false)
	if settings.Signals.Migrations.Enabled {
		row("migration heads", fmt.Sprint(v.State.Migrations), penalised["migrations"])
	}
	row("last tests", v.Tests, penalised["tests"])
	for name, value := range v.State.External {
		row(name, fmt.Sprint(value), false)
	}
	fmt.Fprintln(&out)
	return out.String()
}
