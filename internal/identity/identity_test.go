package identity

import "testing"

// Golden values generated from the original shell implementation via lib.sh's
// buddy_species. They exist so the Go port never silently reassigns a creature
// somebody has already come to recognise as their branch.
func TestForMatchesShellImplementation(t *testing.T) {
	cases := []struct {
		branch  string
		species string
		color   int
	}{
		{"main", "rabbit", 81},
		{"master", "frog", 179},
		{"develop", "axolotl", 187},
		{"trunk", "gecko", 156},
		{"feat/oauth-refresh", "cat", 156},
		{"refactor/auth-guard", "cat", 173},
		{"agent-task-1785874237", "crab", 79},
		{"claude/wiki-updates", "bear", 173},
		{"spike/wasm-build", "bear", 208},
		{"fix/n-plus-one-query", "moth", 213},
		{"chore/bump-deps", "beetle", 81},
		{"docs/install-rewrite", "squid", 187},
		{"a", "mouse", 141},
	}

	for _, testCase := range cases {
		pet := For(testCase.branch)
		if pet.Name != testCase.species || pet.Color != testCase.color {
			t.Errorf("For(%q) = %s/%d, shell gives %s/%d",
				testCase.branch, pet.Name, pet.Color, testCase.species, testCase.color)
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

// Species and colour are hashed separately so two branches landing on the same
// creature still read apart. This pins that they cannot collapse into one hash.
func TestSpeciesAndColorUseDifferentHashes(t *testing.T) {
	catA := For("refactor/auth-guard")
	catB := For("feat/oauth-refresh")
	if catA.Name != catB.Name {
		t.Skip("fixture branches no longer collide on species")
	}
	if catA.Color == catB.Color {
		t.Error("same species and same colour: the two branches are indistinguishable")
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
	if got := Hash("main"); got != Hash("main") {
		t.Fatal("Hash is not stable across calls")
	}
	if Hash("main") == Hash("hue:main") {
		t.Error("branch and hue hashes collide, which would tie colour to species")
	}
}
