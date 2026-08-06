package identity

import (
	"fmt"
	"testing"
)

// Golden values for the current roster. Identity reshuffled exactly once, when
// rarity bands landed: before that a branch picked uniformly from 16 species,
// and afterwards it picks a band first. That was a deliberate one-time change,
// and this table exists so it never happens again by accident.
func TestForIsPinned(t *testing.T) {
	cases := []struct {
		branch  string
		species string
		color   int
		rarity  Rarity
		shiny   bool
	}{
		{"main", "mouse", 81, Common, false},
		{"master", "vole", 179, Common, false},
		{"develop", "vole", 187, Common, false},
		{"trunk", "wren", 156, Common, false},
		{"feat/oauth-refresh", "koi", 156, Rare, false},
		{"refactor/auth-guard", "moth", 173, Common, false},
		{"agent-task-1785874237", "snail", 79, Common, false},
		{"claude/wiki-updates", "gecko", 173, Uncommon, false},
		{"spike/wasm-build", "cat", 208, Common, false},
		{"fix/n-plus-one-query", "owl", 213, Uncommon, false},
		{"chore/bump-deps", "fox", 81, Common, false},
		{"docs/install-rewrite", "seal", 187, Rare, false},
		{"a", "crab", 141, Common, false},
	}

	for _, testCase := range cases {
		pet := For(testCase.branch)
		if pet.Name != testCase.species || pet.Color != testCase.color ||
			pet.Rarity != testCase.rarity || pet.Shiny != testCase.shiny {
			t.Errorf("For(%q) = %s/%d/%s/shiny=%v, want %s/%d/%s/shiny=%v",
				testCase.branch, pet.Name, pet.Color, pet.Rarity, pet.Shiny,
				testCase.species, testCase.color, testCase.rarity, testCase.shiny)
		}
	}
}

func TestForIsDeterministic(t *testing.T) {
	first := For("feat/checkout-flow")
	for attempt := 0; attempt < 100; attempt++ {
		if For("feat/checkout-flow") != first {
			t.Fatal("For is not deterministic")
		}
	}
}

// The weight table is authoritative, not the pool sizes. This is what lets packs
// add creatures to any band without inflating how often that band appears.
func TestRarityMatchesTheWeightTable(t *testing.T) {
	const sample = 20000
	counts := map[Rarity]int{}
	for index := 0; index < sample; index++ {
		counts[For(fmt.Sprintf("feat/ticket-%d-work", index)).Rarity]++
	}
	targets := map[Rarity]float64{
		Common: 60, Uncommon: 25, Rare: 11, Legendary: 3.5, Mythic: 0.5,
	}
	for band, target := range targets {
		got := float64(counts[band]) / float64(sample) * 100
		if got < target-1.5 || got > target+1.5 {
			t.Errorf("%s = %.2f%%, want %.1f%% within 1.5 points", band, got, target)
		}
	}
}

// Adding creatures to a band must not change how often that band is rolled.
func TestPoolSizeDoesNotChangeBandOdds(t *testing.T) {
	pools := CountByRarity()
	if pools[Mythic] >= pools[Common] {
		t.Skip("roster no longer has a small mythic pool to prove the point with")
	}
	const sample = 20000
	mythic := 0
	for index := 0; index < sample; index++ {
		if For(fmt.Sprintf("branch-%d", index)).Rarity == Mythic {
			mythic++
		}
	}
	rate := float64(mythic) / float64(sample) * 100
	if rate > 2 {
		t.Errorf("mythic rate %.2f%% is far above its 0.5%% weight", rate)
	}
}

func TestShinyIsIndependentOfRarity(t *testing.T) {
	const sample = 20000
	shiny, shinyCommon := 0, 0
	for index := 0; index < sample; index++ {
		pet := For(fmt.Sprintf("feat/ticket-%d-work", index))
		if pet.Shiny {
			shiny++
			if pet.Rarity == Common {
				shinyCommon++
			}
		}
	}
	rate := float64(shiny) / float64(sample)
	if rate < 0.004 || rate > 0.013 {
		t.Errorf("shiny rate %.4f, want roughly 1 in 128", rate)
	}
	if shinyCommon == 0 {
		t.Error("no shiny commons appeared, so shininess is not independent of band")
	}
}

func TestRosterIsValid(t *testing.T) {
	if err := Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestKeyDistinguishesShinies(t *testing.T) {
	plain := Pet{Species: Species{Name: "koi"}}
	shiny := Pet{Species: Species{Name: "koi"}, Shiny: true}
	if plain.Key() == shiny.Key() {
		t.Error("a shiny and a plain koi share a den key, so one would overwrite the other")
	}
}

func TestForHandlesEmptyAndUnicodeBranches(t *testing.T) {
	for _, branch := range []string{"", "ünïcøde/brânch", "very/" + string(make([]byte, 300))} {
		pet := For(branch)
		if pet.Name == "" || pet.Prefix == "" {
			t.Errorf("For(%q) produced an unusable pet: %+v", branch, pet)
		}
	}
}

func TestHashIsStable(t *testing.T) {
	if Hash("main") != Hash("main") {
		t.Fatal("Hash is not stable across calls")
	}
	// Species, colour, band and shininess each need their own hash, or they correlate.
	seeds := []string{"main", "hue:main", "band:main", "shiny:main"}
	seen := map[int]bool{}
	for _, seed := range seeds {
		if seen[Hash(seed)] {
			t.Errorf("hash collision across seeds at %q, which would tie two traits together", seed)
		}
		seen[Hash(seed)] = true
	}
}
