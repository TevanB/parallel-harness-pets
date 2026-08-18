package identity

import "testing"

// A place read back from a stored key must be the same place the worktree
// resolved to when it was written, or the visit log would show cities nobody
// ever worked in.
func TestPlaceForKeyAgreesWithPlaceFor(t *testing.T) {
	for _, c := range [][2]string{{"demo", "feat-a"}, {"demo", "feat-b"}, {"other", "feat-a"}} {
		direct := PlaceFor(c[0], c[1])
		viaKey := PlaceForKey(DenKey(c[0], c[1]))
		if direct.Code != viaKey.Code {
			t.Errorf("%s/%s: PlaceFor=%s PlaceForKey=%s", c[0], c[1], direct.Code, viaKey.Code)
		}
	}
}
