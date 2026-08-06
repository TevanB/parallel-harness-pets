package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	now := time.Unix(1786057184, 0)
	written := State{
		Dirty: 5, Unpushed: 1, Behind: 12, Migrations: 2,
		External: map[string]int{"clippy": 3}, Stamp: now,
	}
	if err := Write(dir, "key", written); err != nil {
		t.Fatal(err)
	}
	read, ok := Read(dir, "key")
	if !ok {
		t.Fatal("Read reported no state after Write")
	}
	if read.Dirty != 5 || read.Unpushed != 1 || read.Behind != 12 || read.Migrations != 2 {
		t.Errorf("counts round-tripped as %+v", read)
	}
	if read.External["clippy"] != 3 {
		t.Errorf("external signal lost: %v", read.External)
	}
	if !read.Stamp.Equal(now) {
		t.Errorf("stamp = %v, want %v", read.Stamp, now)
	}
}

func TestReadOnMissingFile(t *testing.T) {
	read, ok := Read(t.TempDir(), "absent")
	if ok {
		t.Error("Read reported state for a file that does not exist")
	}
	if read.External == nil {
		t.Error("External map must be usable even when nothing was read")
	}
}

// A half-written or hand-edited cache must degrade to zeroes rather than leak a
// parse error into somebody's status bar.
func TestGarbageDegradesToZero(t *testing.T) {
	dir := t.TempDir()
	garbage := "dirty=not-a-number\nunpushed=-4\nbehind=\ntorn-lin"
	if err := os.WriteFile(filepath.Join(dir, "key.state"), []byte(garbage), 0o644); err != nil {
		t.Fatal(err)
	}
	read, ok := Read(dir, "key")
	if !ok {
		t.Fatal("a readable file should still report ok")
	}
	if read.Dirty != 0 || read.Unpushed != 0 || read.Behind != 0 {
		t.Errorf("garbage produced %+v, want zeroes", read)
	}
}

func TestTestVerdictDecays(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	if err := WriteTests(dir, "key", "fail", now); err != nil {
		t.Fatal(err)
	}
	if got := ReadTests(dir, "key", now); got != "fail" {
		t.Errorf("fresh verdict = %q, want fail", got)
	}
	// A stale red must not haunt the branch forever.
	if got := ReadTests(dir, "key", now.Add(3*time.Hour)); got != "unknown" {
		t.Errorf("three-hour-old verdict = %q, want unknown", got)
	}
}

func TestStaleness(t *testing.T) {
	now := time.Now()
	if (State{Stamp: now}).Stale(now) {
		t.Error("state written now should not be stale")
	}
	if !(State{Stamp: now.Add(-time.Minute)}).Stale(now) {
		t.Error("a minute-old state should be stale")
	}
}
