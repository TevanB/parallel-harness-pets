package render

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/TevanB/parallel-harness-pets/internal/config"
	"github.com/TevanB/parallel-harness-pets/internal/den"
	"github.com/TevanB/parallel-harness-pets/internal/identity"
	"github.com/TevanB/parallel-harness-pets/internal/score"
)

// rarityHue keeps a band's colour consistent across the party, den and hatch.
func rarityHue(rarity identity.Rarity) string {
	switch rarity {
	case identity.Mythic:
		return "\033[38;5;213m"
	case identity.Legendary:
		return "\033[38;5;141m"
	case identity.Rare:
		return "\033[38;5;81m"
	case identity.Uncommon:
		return "\033[38;5;79m"
	default:
		return "\033[38;5;252m"
	}
}

// Party is the view no single-pet companion can render: every live worktree at
// once, with the worst signal across all of them called out at the bottom.
func Party(views []View, settings config.Config) string {
	if len(views) == 0 {
		return fmt.Sprintf("%s(-.-) no worktrees seen yet. open one.%s\n", dim, reset)
	}
	// Worst first: this is a health dashboard before it is a trophy case, and
	// sorting by branch below keeps the order stable between refreshes.
	sort.Slice(views, func(first, second int) bool {
		if views[first].Score.Hearts != views[second].Score.Hearts {
			return views[first].Score.Hearts < views[second].Score.Hearts
		}
		return views[first].Branch < views[second].Branch
	})

	var out strings.Builder
	fmt.Fprintf(&out, "\n  %spets party%s%s%s%d alive%s\n\n",
		label, reset, strings.Repeat(" ", 34), dim, len(views), reset)

	widestSpecies, widestBranch := 0, 0
	for _, view := range views {
		if width := displayWidth(view.Pet.Label()); width > widestSpecies {
			widestSpecies = width
		}
		if width := displayWidth(view.Branch); width > widestBranch {
			widestBranch = width
		}
	}
	if widestBranch > settings.Display.BranchLabelMax {
		widestBranch = settings.Display.BranchLabelMax
	}

	worst := views[0]
	for _, view := range views {
		body := hue(view.Pet.Color) + pad(view.Body(), 9) + reset
		species := pad(view.Pet.Label(), widestSpecies)
		branch := pad(truncate(view.Branch, settings.Display.BranchLabelMax), widestBranch)

		fmt.Fprintf(&out, "  %s %s%s%s %s%-5s%s %s%s%s  %s",
			body,
			hue(view.Pet.Color), species, reset,
			rarityHue(view.Pet.Rarity), view.Pet.Rarity.Stars(), reset,
			label, branch, reset,
			view.hearts())
		if flags := view.Flags(settings); len(flags) > 0 {
			fmt.Fprintf(&out, "  %s%s%s", warn, strings.Join(flags, " "), reset)
		}
		fmt.Fprintln(&out)
	}

	if len(worst.Score.Penalties) > 0 {
		// A signal can charge twice, once per threshold, but naming it twice
		// reads as a bug rather than as emphasis.
		seen := map[string]bool{}
		reasons := make([]string, 0, len(worst.Score.Penalties))
		for _, penalty := range worst.Score.Penalties {
			if seen[penalty.Signal] {
				continue
			}
			seen[penalty.Signal] = true
			reasons = append(reasons, penalty.Signal)
		}
		fmt.Fprintf(&out, "\n  %sworst:%s %s%s%s %s·%s %s%s%s\n",
			dim, reset, hue(worst.Pet.Color), worst.Pet.Name, reset,
			dim, reset, warn, strings.Join(reasons, ", "), reset)
	}
	fmt.Fprintln(&out)
	return out.String()
}

// Den is the collection view: what you have hatched, and what is still missing.
func Den(collection den.Den) string {
	var out strings.Builder
	progress := collection.Completion()
	have, total := 0, 0
	for _, band := range progress {
		have += band.Have
		total += band.Total
	}

	fmt.Fprintf(&out, "\n  %spets den%s%s%d/%d species · %d shiny%s\n\n",
		label, reset, strings.Repeat(" ", 22), have, total, collection.ShinyCount(), reset)

	const width = 20
	for _, band := range progress {
		filled := 0
		if band.Total > 0 {
			filled = band.Have * width / band.Total
		}
		fmt.Fprintf(&out, "  %s%-11s%s %s%s%s%s%s  %s%d/%d%s\n",
			rarityHue(band.Rarity), strings.ToUpper(band.Rarity.String()), reset,
			rarityHue(band.Rarity), strings.Repeat("█", filled),
			dim, strings.Repeat("░", width-filled), reset,
			dim, band.Have, band.Total, reset)
	}

	if latest := collection.Latest(1); len(latest) > 0 {
		entry := latest[0]
		fmt.Fprintf(&out, "\n  %slatest%s  %s%s%s  %s%s%s  %s%s · %s%s\n",
			dim, reset,
			label, entry.Species, reset,
			rarityHue(rarityByName(entry.Rarity)), entry.Rarity, reset,
			dim, entry.Branch, humanAge(entry.FirstSeen), reset)
	}
	fmt.Fprintln(&out)
	return out.String()
}

// Hatch is the one animated moment in the project: a branch never opened before.
//
// collected is false when the worktree has no commit yet, in which case the
// creature is shown but has not entered the den.
func Hatch(pet identity.Pet, have, total int, collected bool) string {
	var out strings.Builder
	tint := rarityHue(pet.Rarity)
	fmt.Fprintf(&out, "\n       %s.-\"\"\"-.%s\n", dim, reset)
	fmt.Fprintf(&out, "      %s/  . .  \\%s       %sa branch you have not opened before%s\n", dim, reset, dim, reset)
	fmt.Fprintf(&out, "     %s|  .   . |%s\n", dim, reset)
	fmt.Fprintf(&out, "      %s\\  ...  /%s\n", dim, reset)
	fmt.Fprintf(&out, "       %s'-...-'%s\n\n", dim, reset)
	fmt.Fprintf(&out, "       %s%s%s\n\n", hue(pet.Color), pet.Prefix+score.Face(score.Max)+pet.Suffix, reset)
	fmt.Fprintf(&out, "       %s%s%s  %s%s %s%s\n",
		hue(pet.Color), pet.Label(), reset,
		tint, pet.Rarity.Stars(), strings.ToUpper(pet.Rarity.String()), reset)
	if collected {
		fmt.Fprintf(&out, "       %sfirst hatch · %d of %d in your den%s\n\n", dim, have, total, reset)
	} else {
		fmt.Fprintf(&out, "       %sfirst hatch · commit here to keep it%s\n\n", dim, reset)
	}
	return out.String()
}

func rarityByName(name string) identity.Rarity {
	for band := identity.Common; band <= identity.Mythic; band++ {
		if band.String() == name {
			return band
		}
	}
	return identity.Common
}

func humanAge(when time.Time) string {
	elapsed := time.Since(when)
	switch {
	case elapsed < time.Minute:
		return "just now"
	case elapsed < time.Hour:
		return fmt.Sprintf("%dm ago", int(elapsed.Minutes()))
	case elapsed < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(elapsed.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(elapsed.Hours()/24))
	}
}
