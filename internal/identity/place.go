package identity

import "fmt"

// A Place is the den a worktree lives in.
//
// Where a Pet is rolled per agent and is gone when that agent stops, the Place
// is stable: the same repo and worktree name resolve to the same city on every
// machine, with nothing stored. Codes are IATA so a contributor adding a city
// has an external standard to follow rather than an argument to have.
//
// There was a decorative glyph beside the code once, and it is not coming back.
// Sixteen single characters picked for sixteen skylines never read as one
// family; Dubai's arrow collided with the arrow already meaning "unpushed"; two
// cities silently shared a glyph because only codes were checked for
// uniqueness; and half of them were East Asian Ambiguous, so they were narrow
// here and double-width in a CJK terminal, misaligning every column after them.
// The drawing does that job, in the surfaces that have room for it.
type Place struct {
	Code string
	Name string
	// Art is drawn only where there is room: the card and the den, never the
	// status line. Pure ASCII, so a byte count is a column count.
	Art [3]string
}

// ArtWidth is the column budget every city drawing shares, so a den lays out in
// a grid without measuring anything.
const ArtWidth = 10

var places = []Place{
	{"SFO", "San Francisco", [3]string{
		`  /\  /\  `,
		` /||__||\ `,
		`~~~~~~~~~~`}},
	{"TYO", "Tokyo", [3]string{
		`__________`,
		` _|____|_ `,
		`  |    |  `}},
	{"NYC", "New York", [3]string{
		`     |    `,
		`   _|_|_  `,
		` _|_|_|_|_`}},
	{"PAR", "Paris", [3]string{
		`    /\    `,
		`   /||\   `,
		`  /_||_\  `}},
	{"LON", "London", [3]string{
		`    ^     `,
		`   |o|    `,
		` __|_|____`}},
	{"CAI", "Cairo", [3]string{
		`  /\      `,
		` /  \ /\  `,
		`/____\/__\`}},
	{"SYD", "Sydney", [3]string{
		`   _  _   `,
		`  / \/ \  `,
		` /______\ `}},
	{"BER", "Berlin", [3]string{
		` ________ `,
		` |||||||| `,
		` |_||||_| `}},
	{"SEA", "Seattle", [3]string{
		`    |     `,
		`   /_\    `,
		`   |_|    `}},
	{"AMS", "Amsterdam", [3]string{
		` _  _  _  `,
		`|_||_||_| `,
		`|_||_||_| `}},
	{"RIO", "Rio de Janeiro", [3]string{
		`   \|/    `,
		`    |     `,
		`   /_\    `}},
	{"IST", "Istanbul", [3]string{
		` |  __  | `,
		` | /  \ | `,
		` |/____\| `}},
	{"DXB", "Dubai", [3]string{
		`    |     `,
		`   /|\    `,
		`  /_|_\   `}},
	{"SIN", "Singapore", [3]string{
		`  ______  `,
		` || || || `,
		` || || || `}},
	{"HKG", "Hong Kong", [3]string{
		`   _   _  `,
		`  | |_| | `,
		` |_|_|_|_|`}},
	{"MEX", "Mexico City", [3]string{
		`    __    `,
		`   /__\   `,
		`  /____\  `}},
}

// PlaceFor resolves the den from the repository and the worktree's own name.
//
// Deliberately not the branch: a den is a location, and checking out a different
// branch inside a worktree should not relocate it. Deliberately not the absolute
// path either, which would bake one machine's directory layout into identity.
//
// The repository owner is deliberately absent. A harness payload carries it but
// `pets card` has no payload, and including it made one worktree resolve to two
// different cities depending on which surface asked.
func PlaceFor(repo, worktree string) Place {
	return places[Hash(DenKey(repo, worktree))%len(places)]
}

// DenKey names a den for storage and grouping. One string, so agents can be
// grouped by the den they are in without any of them consulting git.
func DenKey(repo, worktree string) string {
	return "place:" + repo + "#" + worktree
}

// AllPlaces returns the full roster, for the den's completion view.
func AllPlaces() []Place {
	roster := make([]Place, len(places))
	copy(roster, places)
	return roster
}

// ValidatePlaces keeps the roster mechanically correct so "add a city" stays a
// one-line contribution that a test can accept or reject without taste.
func ValidatePlaces() error {
	seen := map[string]bool{}
	for _, place := range places {
		if len(place.Code) != 3 {
			return fmt.Errorf("place %q: code must be exactly 3 characters", place.Code)
		}
		for _, char := range place.Code {
			if char < 'A' || char > 'Z' {
				return fmt.Errorf("place %q: code must be uppercase A-Z", place.Code)
			}
		}
		if seen[place.Code] {
			return fmt.Errorf("duplicate place %q", place.Code)
		}
		seen[place.Code] = true
		for row, line := range place.Art {
			// Byte length is the column count only while the art stays ASCII,
			// which is also what keeps it aligned in every terminal.
			for _, char := range line {
				if char > 127 {
					return fmt.Errorf("place %q art row %d: must be ASCII", place.Code, row)
				}
			}
			if len(line) != ArtWidth {
				return fmt.Errorf("place %q art row %d: must be exactly %d characters, got %d",
					place.Code, row, ArtWidth, len(line))
			}
		}
	}
	return nil
}
