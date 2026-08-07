package render

import (
	"strings"
	"testing"

	"github.com/TevanB/parallel-harness-pets/internal/config"
	"github.com/TevanB/parallel-harness-pets/internal/identity"
	"github.com/TevanB/parallel-harness-pets/internal/score"
	"github.com/TevanB/parallel-harness-pets/internal/state"
)

// A terminal renders an emoji in two cells but Go counts it as one rune, so
// padding from a rune count misaligns every column after a shiny's sparkle.
func TestDisplayWidthCountsTerminalCells(t *testing.T) {
	cases := []struct {
		text  string
		width int
	}{
		{"gecko", 5},
		{"gecko ✨", 8},
		{"", 0},
		{">(•ᴗ•)<", 7},
		{"~(•ᴗ•)~", 7},
		{"✦✦✦", 3},
		{"♥♥♥♡♡", 5},
		{"△↑↓⑂✗", 5},
		{"████░░", 6},
		{"feat/機能追加", 5 + 8},
	}
	for _, testCase := range cases {
		if got := displayWidth(testCase.text); got != testCase.width {
			t.Errorf("displayWidth(%q) = %d, want %d", testCase.text, got, testCase.width)
		}
	}
}

func TestPadFillsToCellWidth(t *testing.T) {
	if got := displayWidth(pad("gecko ✨", 10)); got != 10 {
		t.Errorf("pad to 10 produced %d cells", got)
	}
	if got := displayWidth(pad("ant", 10)); got != 10 {
		t.Errorf("pad to 10 produced %d cells", got)
	}
	// Padding must never truncate.
	if !strings.HasPrefix(pad("a-very-long-branch-name", 3), "a-very-long-branch-name") {
		t.Error("pad truncated instead of leaving the text alone")
	}
}

func view(branch string, hearts int, shiny bool) View {
	pet := identity.For(branch)
	pet.Shiny = shiny
	return View{
		Pet: pet, Branch: branch, HasState: true,
		Score: score.Result{Hearts: hearts},
		State: state.State{External: map[string]int{}},
	}
}

// Every party row must start its columns at the same cell, shiny or not.
func TestPartyColumnsAlign(t *testing.T) {
	views := []View{
		view("feat/oauth-flow", 4, true),
		view("chore/deps-bump", 4, false),
		view("spike/graphql", 1, false),
	}
	rendered := stripANSI(Party(views, config.Default()))

	var starts []int
	for _, line := range strings.Split(rendered, "\n") {
		index := strings.Index(line, "✦")
		if index < 0 || strings.Contains(line, "pets party") {
			continue
		}
		starts = append(starts, displayWidth(line[:index]))
	}
	if len(starts) < 3 {
		t.Fatalf("expected 3 creature rows, found %d", len(starts))
	}
	for _, start := range starts[1:] {
		if start != starts[0] {
			t.Errorf("rarity column starts at %v across rows, want them equal", starts)
			break
		}
	}
}

// Worst first, because this is a health dashboard before it is a trophy case.
func TestPartySortsWorstFirst(t *testing.T) {
	views := []View{
		view("chore/deps-bump", 5, false),
		view("spike/graphql", 1, false),
		view("feat/oauth-flow", 3, false),
	}
	rendered := stripANSI(Party(views, config.Default()))
	positions := []int{
		strings.Index(rendered, "spike/graphql"),
		strings.Index(rendered, "feat/oauth-flow"),
		strings.Index(rendered, "chore/deps-bump"),
	}
	for index := 1; index < len(positions); index++ {
		if positions[index] < positions[index-1] {
			t.Errorf("party is not sorted worst-first: %v", positions)
			break
		}
	}
}

func TestPartyWithNoWorktrees(t *testing.T) {
	if out := Party(nil, config.Default()); !strings.Contains(out, "no worktrees") {
		t.Errorf("empty party rendered %q", out)
	}
}

// The renderer must never emit a bare newline-free blob or leave colour on.
func TestStatuslineResetsColour(t *testing.T) {
	rendered := Statusline(view("feat/oauth-flow", 4, false), config.Default())
	if !strings.HasSuffix(rendered, reset) {
		t.Error("status line does not end with a colour reset, so it would bleed into the bar")
	}
	if strings.Contains(rendered, "\n") {
		t.Error("status line contains a newline")
	}
}

func stripANSI(text string) string {
	var out strings.Builder
	inEscape := false
	for _, char := range text {
		switch {
		case char == 0x1b:
			inEscape = true
		case inEscape && char == 'm':
			inEscape = false
		case !inEscape:
			out.WriteRune(char)
		}
	}
	return out.String()
}
