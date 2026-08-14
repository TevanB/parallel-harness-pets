// Package agents is the register of who is currently working where.
//
// A pet belongs to an agent, not to a directory, so something has to remember
// which agent is in which den and when it was last heard from. That is this.
//
// Every agent writes only its own file, named for its session, so two agents in
// one worktree never contend for a write and no locking is needed. The register
// is disposable: a missing or torn file costs a row in a list, never an error.
package agents

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
	// Heartbeat is how often an agent bothers to re-record itself. The status
	// line calls in roughly once a second in every session; writing that often
	// would put pointless I/O on the one path that must stay cheap.
	Heartbeat = 10 * time.Second
	// StaleAfter is when an agent stops counting as live. An agent that exits
	// never gets to say so, so silence is the only signal available.
	StaleAfter = 90 * time.Second
	// Forgotten is when a silent agent stops being listed at all, so a machine
	// left running for a week does not accumulate ghosts.
	Forgotten = 12 * time.Hour
)

// Record is one agent, pinned to the den it is working in.
type Record struct {
	// Session is the harness's own id for this agent. It is the key the pet is
	// derived from, which is what pins a creature to an agent.
	Session string
	// Den is the repo-and-worktree key, so agents group without consulting git.
	Den string
	// Root is the worktree path, kept for display. It is deliberately not used
	// to decide whether an agent still exists: a worktree can be deleted out
	// from under a session that is very much still running.
	Root string
	// Name is the harness's human label for the session. It is what makes a
	// list of agents readable rather than a column of UUIDs.
	Name   string
	Branch string
	Seen   time.Time
	// Since is when this agent last changed den. Seen says the agent is alive;
	// Since says how long it has been here, which is a different question once
	// an agent can move between worktrees.
	Since time.Time
}

func (r Record) Stale(now time.Time) bool { return now.Sub(r.Seen) >= StaleAfter }

// JustArrived reports an agent that has only recently moved into this den, so a
// listing can show travel rather than presenting a newcomer as a fixture.
func (r Record) JustArrived(now time.Time) bool {
	return !r.Since.IsZero() && now.Sub(r.Since) < time.Minute
}

// Label is what to show a human: the session's own name when it has one, and a
// short slice of the id when it does not, so the column is never blank.
func (r Record) Label() string {
	if name := strings.TrimSpace(r.Name); name != "" {
		return name
	}
	if len(r.Session) > 8 {
		return r.Session[:8]
	}
	return r.Session
}

func dir(stateDir string) string { return filepath.Join(stateDir, "agents") }

// fileName keeps a session id safe to use as a filename on every platform,
// for the same reason the worktree cache key does.
func fileName(session string) string {
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
	return flat.String() + ".agent"
}

// Touch records an agent as alive, skipping the write when the existing record
// is recent enough. The status line calls this once a second per session.
func Touch(stateDir string, record Record, now time.Time) error {
	if record.Session == "" {
		return nil
	}
	path := filepath.Join(dir(stateDir), fileName(record.Session))
	record.Since = now
	if existing, err := read(path); err == nil {
		if existing.Den == record.Den {
			// Same den, so the arrival time carries over rather than being reset
			// by every heartbeat.
			record.Since = existing.Since
			if now.Sub(existing.Seen) < Heartbeat {
				return nil
			}
		}
		// A changed den is written immediately whatever the heartbeat says: the
		// move is the interesting event, and delaying it loses the arrival time.
	}
	if err := os.MkdirAll(dir(stateDir), 0o755); err != nil {
		return err
	}
	record.Seen = now
	var out strings.Builder
	fmt.Fprintf(&out, "session=%s\nden=%s\nroot=%s\nbranch=%s\nname=%s\nts=%d\nsince=%d\n",
		record.Session, record.Den, record.Root, record.Branch,
		// A newline in a session name would forge a second field on read.
		strings.ReplaceAll(record.Name, "\n", " "), record.Seen.Unix(), record.Since.Unix())
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, []byte(out.String()), 0o644); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

// Beat records an agent as still alive without claiming to know where it is.
//
// An agent that steps outside a git worktree is still running, and coupling
// registration to location made it vanish from every listing after 90 seconds.
// The last known den is preserved rather than cleared, so the agent stays where
// you last saw it instead of blinking out and back.
func Beat(stateDir, session string, now time.Time) error {
	if session == "" {
		return nil
	}
	existing, err := read(filepath.Join(dir(stateDir), fileName(session)))
	if err != nil {
		// Nothing to keep alive: an agent that has never been in a worktree has
		// no den to show and nothing worth recording.
		return nil
	}
	return Touch(stateDir, existing, now)
}

func read(path string) (Record, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Record{}, err
	}
	record := Record{}
	for _, line := range strings.Split(string(data), "\n") {
		name, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		switch name {
		case "session":
			record.Session = value
		case "den":
			record.Den = value
		case "root":
			record.Root = value
		case "branch":
			record.Branch = value
		case "name":
			record.Name = value
		case "ts":
			seconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
			if err != nil {
				return Record{}, err
			}
			record.Seen = time.Unix(seconds, 0)
		case "since":
			seconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
			if err != nil {
				return Record{}, err
			}
			record.Since = time.Unix(seconds, 0)
		}
	}
	if record.Session == "" {
		return Record{}, fmt.Errorf("no session in %s", path)
	}
	return record, nil
}

// All returns every agent still worth showing, most recently seen first.
//
// Silence is the only thing that retires an agent. Pruning on a missing
// directory used to delete live agents whose worktree had just been removed,
// which is a rendering concern rather than an existence one.
func All(stateDir string, now time.Time) []Record {
	entries, err := os.ReadDir(dir(stateDir))
	if err != nil {
		return nil
	}
	var found []Record
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".agent") {
			continue
		}
		path := filepath.Join(dir(stateDir), entry.Name())
		record, err := read(path)
		if err != nil {
			os.Remove(path)
			continue
		}
		if now.Sub(record.Seen) >= Forgotten {
			os.Remove(path)
			continue
		}
		found = append(found, record)
	}
	sort.Slice(found, func(first, second int) bool {
		return found[first].Seen.After(found[second].Seen)
	})
	return found
}

// Representative picks the agent that should stand for a den.
//
// A den always shows a creature, but when somebody is actually working there
// the creature ought to be theirs rather than one derived from the branch. A
// live agent always outranks a silent one, however recently the silent one
// spoke, because a den's face should belong to whoever is still in it.
//
// It does not assume the caller sorted anything, so the rule survives a change
// to how the register is ordered.
func Representative(records []Record, now time.Time) (Record, bool) {
	var best Record
	found := false
	for _, record := range records {
		switch {
		case !found:
			best, found = record, true
		case best.Stale(now) && !record.Stale(now):
			best = record
		case best.Stale(now) == record.Stale(now) && record.Seen.After(best.Seen):
			best = record
		}
	}
	return best, found
}

// InDen returns the agents working in one den, most recently seen first.
func InDen(stateDir, den string, now time.Time) []Record {
	var found []Record
	for _, record := range All(stateDir, now) {
		if record.Den == den {
			found = append(found, record)
		}
	}
	return found
}
