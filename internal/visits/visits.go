// Package visits is a den's memory of everyone who has worked in it.
//
// The agent register answers who is here now, and deliberately forgets: when an
// agent moves, its single record is overwritten in place, which is what stops a
// ghost being left behind. That same property means a den could never say who
// had passed through. This is the additive half.
//
// One file per den-and-session pair, so no two writers ever share a file and no
// locking is needed, the same trick the register uses. A den's visitors are the
// files sharing its prefix.
package visits

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Visit is one agent's history with one den.
type Visit struct {
	Session string
	Den     string
	First   time.Time
	Last    time.Time
}

func dir(stateDir string) string { return filepath.Join(stateDir, "visits") }

// prefix folds a den key into something usable as a filename. The key contains
// a colon and a hash, neither of which is safe on every platform.
func prefix(den string) string {
	digest := fnv.New32a()
	digest.Write([]byte(den))
	return fmt.Sprintf("%08x", digest.Sum32())
}

func safe(session string) string {
	var flat strings.Builder
	for _, char := range session {
		switch {
		case char >= 'a' && char <= 'z', char >= 'A' && char <= 'Z',
			char >= '0' && char <= '9', char == '-':
			flat.WriteRune(char)
		default:
			flat.WriteRune('_')
		}
	}
	return flat.String()
}

func path(stateDir, den, session string) string {
	return filepath.Join(dir(stateDir), prefix(den)+"-"+safe(session)+".visit")
}

// Record notes that a session has been in a den, keeping the first arrival and
// moving the last. Called on the render path, so it writes only when the entry
// is new or the day has moved on.
func Record(stateDir, den, session string, now time.Time) error {
	if den == "" || session == "" {
		return nil
	}
	file := path(stateDir, den, session)
	visit := Visit{Session: session, Den: den, First: now, Last: now}
	if existing, err := read(file); err == nil {
		// First arrival is the fact worth keeping; rewriting it on every render
		// would turn a history into a clock.
		visit.First = existing.First
		if now.Sub(existing.Last) < time.Hour {
			return nil
		}
	}
	if err := os.MkdirAll(dir(stateDir), 0o755); err != nil {
		return err
	}
	body := fmt.Sprintf("session=%s\nden=%s\nfirst=%d\nlast=%d\n",
		visit.Session, visit.Den, visit.First.Unix(), visit.Last.Unix())
	temporary := file + ".tmp"
	if err := os.WriteFile(temporary, []byte(body), 0o644); err != nil {
		return err
	}
	return os.Rename(temporary, file)
}

func read(file string) (Visit, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return Visit{}, err
	}
	visit := Visit{}
	for _, line := range strings.Split(string(data), "\n") {
		name, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		switch name {
		case "session":
			visit.Session = value
		case "den":
			visit.Den = value
		case "first", "last":
			seconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
			if err != nil {
				return Visit{}, err
			}
			if name == "first" {
				visit.First = time.Unix(seconds, 0)
			} else {
				visit.Last = time.Unix(seconds, 0)
			}
		}
	}
	if visit.Session == "" {
		return Visit{}, fmt.Errorf("no session in %s", file)
	}
	return visit, nil
}

// ForDen returns everyone who has ever worked in a den, first arrival first, so
// the list reads as a history rather than as a leaderboard.
func ForDen(stateDir, den string) []Visit {
	entries, err := os.ReadDir(dir(stateDir))
	if err != nil {
		return nil
	}
	wanted := prefix(den) + "-"
	var found []Visit
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), wanted) || !strings.HasSuffix(entry.Name(), ".visit") {
			continue
		}
		visit, err := read(filepath.Join(dir(stateDir), entry.Name()))
		if err != nil {
			continue
		}
		// The prefix is a hash, so confirm the den rather than trusting a
		// collision to be impossible.
		if visit.Den != den {
			continue
		}
		found = append(found, visit)
	}
	sort.Slice(found, func(first, second int) bool {
		return found[first].First.Before(found[second].First)
	})
	return found
}
