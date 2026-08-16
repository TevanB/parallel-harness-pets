// Package lock provides directory-based locks that clean up after crashed owners.
//
// A directory is used because creation is atomic across processes, and because
// a file left behind by a killed process looks identical to a live one. The
// stale timeout is what makes the lock self-healing.
package lock

import (
	"os"
	"time"
)

// TryAcquire creates a directory-based lock. If a lock already exists but is
// older than stale, it is removed and acquisition is retried once.
func TryAcquire(path string, stale time.Duration) bool {
	if os.Mkdir(path, 0o755) == nil {
		return true
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if time.Since(info.ModTime()) <= stale {
		return false
	}
	_ = os.Remove(path)
	return os.Mkdir(path, 0o755) == nil
}

// WaitAcquire tries to acquire a lock, breaking stale ones and retrying for up
// to about a second.
func WaitAcquire(path string, stale time.Duration) bool {
	for attempt := 0; attempt < 50; attempt++ {
		if TryAcquire(path, stale) {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}
