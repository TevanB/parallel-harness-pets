// Package state is the disposable cache between the probe and the renderer.
//
// Nothing here is precious: a torn or hand-edited file must degrade to zeroes
// rather than leak a parse error into somebody's status bar.
package state

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// A verdict older than this says nothing about the code as it stands now.
	testVerdictLifetime = 2 * time.Hour
	// How long cached git counts are trusted before the renderer refreshes them.
	StaleAfter = 15 * time.Second
)

type State struct {
	Dirty      int
	Unpushed   int
	Behind     int
	Migrations int
	External   map[string]int
	Stamp      time.Time
	// Root is stored rather than decoded from the cache key, because a path
	// containing an underscore cannot be recovered from the flattened key.
	Root   string
	Branch string
}

func (s State) Stale(now time.Time) bool {
	return now.Sub(s.Stamp) >= StaleAfter
}

func atoi(text string) int {
	value, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func Read(dir, key string) (State, bool) {
	loaded := State{External: map[string]int{}}
	data, err := os.ReadFile(filepath.Join(dir, key+".state"))
	if err != nil {
		return loaded, false
	}
	for _, line := range strings.Split(string(data), "\n") {
		name, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		switch name {
		case "dirty":
			loaded.Dirty = atoi(value)
		case "unpushed":
			loaded.Unpushed = atoi(value)
		case "behind":
			loaded.Behind = atoi(value)
		case "migrations":
			loaded.Migrations = atoi(value)
		case "ts":
			loaded.Stamp = time.Unix(int64(atoi(value)), 0)
		case "root":
			loaded.Root = value
		case "branch":
			loaded.Branch = value
		default:
			loaded.External[name] = atoi(value)
		}
	}
	return loaded, true
}

func Write(dir, key string, value State) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "dirty=%d\nunpushed=%d\nbehind=%d\nmigrations=%d\n",
		value.Dirty, value.Unpushed, value.Behind, value.Migrations)
	names := make([]string, 0, len(value.External))
	for name := range value.External {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(&builder, "%s=%d\n", name, value.External[name])
	}
	fmt.Fprintf(&builder, "root=%s\nbranch=%s\nts=%d\n", value.Root, value.Branch, value.Stamp.Unix())
	return replace(filepath.Join(dir, key+".state"), builder.String())
}

// All returns every live worktree the cache knows about, newest first.
//
// It deduplicates by root and prunes entries whose directory is gone. Without
// that, one worktree can appear several times: a deleted worktree lingers
// forever, and any change to how keys are built leaves the same root recorded
// under two filenames, listing it twice with contradictory state.
func All(dir string) []State {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	newest := map[string]State{}
	files := map[string]string{}
	for _, entry := range entries {
		name, isState := strings.CutSuffix(entry.Name(), ".state")
		if !isState {
			continue
		}
		loaded, ok := Read(dir, name)
		if !ok {
			continue
		}
		if info, statErr := os.Stat(loaded.Root); loaded.Root == "" || statErr != nil || !info.IsDir() {
			// The cache is disposable, so an entry pointing nowhere is simply removed.
			os.Remove(filepath.Join(dir, entry.Name()))
			continue
		}
		// Two files for one root means a superseded key scheme. Keep the newest
		// and delete the loser, so the duplicate heals instead of lingering.
		if existing, seen := newest[loaded.Root]; seen {
			if !loaded.Stamp.After(existing.Stamp) {
				os.Remove(filepath.Join(dir, entry.Name()))
				continue
			}
			os.Remove(filepath.Join(dir, files[loaded.Root]))
		}
		newest[loaded.Root] = loaded
		files[loaded.Root] = entry.Name()
	}
	found := make([]State, 0, len(newest))
	for _, loaded := range newest {
		found = append(found, loaded)
	}
	sort.Slice(found, func(first, second int) bool {
		return found[first].Stamp.After(found[second].Stamp)
	})
	return found
}

// Test verdicts live in their own file so the probe and the tool hook never
// race on one another's writes.
func ReadTests(dir, key string, now time.Time) string {
	data, err := os.ReadFile(filepath.Join(dir, key+".tests"))
	if err != nil {
		return "unknown"
	}
	result, stamp := "unknown", int64(0)
	for _, line := range strings.Split(string(data), "\n") {
		name, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		switch name {
		case "result":
			result = strings.TrimSpace(value)
		case "ts":
			stamp = int64(atoi(value))
		}
	}
	if now.Sub(time.Unix(stamp, 0)) > testVerdictLifetime {
		return "unknown"
	}
	return result
}

func WriteTests(dir, key, result string, now time.Time) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return replace(filepath.Join(dir, key+".tests"),
		fmt.Sprintf("result=%s\nts=%d\n", result, now.Unix()))
}

// replace writes through a temporary file so a reader never sees a half-written cache.
func replace(path, contents string) error {
	temporary := fmt.Sprintf("%s.%d", path, os.Getpid())
	if err := os.WriteFile(temporary, []byte(contents), 0o644); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}
