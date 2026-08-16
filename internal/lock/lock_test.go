package lock

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTryAcquireCreatesAndRemovesLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")
	if !TryAcquire(path, time.Minute) {
		t.Fatal("could not acquire fresh lock")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("lock directory was not created: %v", err)
	}
	if TryAcquire(path, time.Minute) {
		t.Fatal("acquired the same lock twice while it was fresh")
	}
	_ = os.Remove(path)
	if !TryAcquire(path, time.Minute) {
		t.Fatal("could not re-acquire lock after removing it")
	}
}

func TestTryAcquireBreaksStaleLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stale.lock")
	if !TryAcquire(path, time.Minute) {
		t.Fatal("could not acquire lock")
	}
	// Roll the modification time back so the lock looks abandoned.
	old := time.Now().Add(-2 * time.Minute)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	if !TryAcquire(path, time.Second) {
		t.Fatal("stale lock was not broken")
	}
}
