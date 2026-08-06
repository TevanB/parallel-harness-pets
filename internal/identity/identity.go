// Package identity derives a worktree's creature from its branch name alone.
//
// Nothing here touches disk or network: the same branch yields the same creature
// on every machine, forever, with nothing persisted.
package identity

// Species is one creature: a name and the two frame fragments bracketing its face.
type Species struct {
	Name   string
	Prefix string
	Suffix string
}

var species = []Species{
	{"otter", "(", ")"},
	{"cat", "/", "\\"},
	{"fox", "{", "}"},
	{"frog", "[", "]"},
	{"bat", "^", "^"},
	{"rabbit", "(\\", "/)"},
	{"moth", "<", ">"},
	{"axolotl", "~(", ")~"},
	{"gecko", "-(", ")-"},
	{"owl", "((", "))"},
	{"mouse", ".(", ")."},
	{"squid", "*(", ")*"},
	{"crab", ">(", ")<"},
	{"bear", "o(", ")o"},
	{"beetle", "+(", ")+"},
	{"koi", "=(", ")="},
}

// Hashed independently of species so two live branches on the same creature still read apart.
var palette = []int{173, 213, 208, 114, 141, 203, 187, 218, 79, 179, 81, 156}

// Pet is a fully resolved creature, ready to render.
type Pet struct {
	Species
	Color int
}

// Hash folds a string into a stable integer, matching the original shell implementation.
//
// It walks bytes rather than runes so the result never depends on the caller's locale.
func Hash(text string) int {
	value := 7
	for index := 0; index < len(text); index++ {
		value = (value*31 + int(text[index])) % 1000003
	}
	return value
}

// For returns the creature belonging to a branch.
func For(branch string) Pet {
	return Pet{
		Species: species[Hash(branch)%len(species)],
		Color:   palette[Hash("hue:"+branch)%len(palette)],
	}
}

// Count reports how many species exist, for tests and the eventual collection view.
func Count() int {
	return len(species)
}
