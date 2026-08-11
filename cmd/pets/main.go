// Command pets renders a per-worktree creature and the branch hygiene behind its mood.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/TevvvB/parallel-harness-pets/internal/config"
	"github.com/TevvvB/parallel-harness-pets/internal/gitrepo"
	"github.com/TevvvB/parallel-harness-pets/internal/identity"
	"github.com/TevvvB/parallel-harness-pets/internal/render"
	"github.com/TevvvB/parallel-harness-pets/internal/score"
	"github.com/TevvvB/parallel-harness-pets/internal/signal"
	"github.com/TevvvB/parallel-harness-pets/internal/state"
	"github.com/TevvvB/parallel-harness-pets/internal/verdict"
)

var version = "dev"

const usage = `pets - a creature for every worktree

  pets render [--format=statusline|tmux|title|json]  render for a surface
  pets party [--all]                                 every live worktree at once
  pets den                                           the collection
  pets card [path]                                   full readout
  pets probe <path>                                  refresh one worktree's cache
  pets hatch                                         session-start hook, reads JSON on stdin
  pets record                                        tool-use hook, reads JSON on stdin
  pets quip                                          stop hook, reads JSON on stdin
  pets install [--harness=claude|codex|tmux|shell|all]
  pets uninstall
  pets version
`

func main() {
	os.Exit(dispatch(os.Args[1:]))
}

// dispatch routes a command and returns the process exit code.
//
// Asking for help is a success and prints to stdout; an unknown command is a
// failure and prints to stderr. Conflating the two makes pets --help exit 1,
// which passes unnoticed interactively and fails every script that runs it.
func dispatch(args []string) int {
	if len(args) == 0 {
		fmt.Print(usage)
		return 0
	}
	switch args[0] {
	case "help", "--help", "-h":
		fmt.Print(usage)
		return 0
	case "render":
		renderCommand(args[1:])
	case "party":
		partyCommand(args[1:])
	case "den":
		denCommand()
	case "card":
		cardCommand(args[1:])
	case "probe":
		probeCommand(args[1:])
	case "hatch":
		hatchCommand()
	case "record":
		recordCommand()
	case "quip":
		quipCommand()
	case "install", "uninstall":
		installCommand(args[0], args[1:])
	case "version", "--version", "-v":
		fmt.Println(version)
	default:
		fmt.Fprintf(os.Stderr, "pets: unknown command %q\n\n%s", args[0], usage)
		return 1
	}
	return 0
}

// hookPayload is the shape both Claude Code and Codex pipe into a hook.
type hookPayload struct {
	Cwd       string `json:"cwd"`
	Workspace struct {
		CurrentDir string `json:"current_dir"`
	} `json:"workspace"`
	Model struct {
		DisplayName string `json:"display_name"`
	} `json:"model"`
	ToolInput struct {
		Command string `json:"command"`
	} `json:"tool_input"`
	ToolResponse json.RawMessage `json:"tool_response"`
}

// stdinDeadline bounds how long a surface may take to hand over its payload.
//
// A harness that pipes JSON writes it immediately and closes, so this is never
// reached. A surface that does not, such as tmux running us through #(...), can
// hand us an inherited stdin that never closes, and an unbounded read there
// hangs the status bar forever.
const stdinDeadline = 150 * time.Millisecond

// readPayload parses harness JSON from stdin, tolerating an absent, empty, or
// never-closing one.
func readPayload() hookPayload {
	var payload hookPayload
	info, err := os.Stdin.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice != 0 {
		return payload
	}
	return parsePayload(os.Stdin, stdinDeadline)
}

// parsePayload is readPayload's testable core: it gives up rather than blocking
// forever on a reader that never reaches EOF.
func parsePayload(source io.Reader, deadline time.Duration) hookPayload {
	var payload hookPayload
	received := make(chan []byte, 1)
	go func() {
		data, err := io.ReadAll(source)
		if err != nil {
			received <- nil
			return
		}
		received <- data
	}()
	select {
	case data := <-received:
		if len(data) > 0 {
			json.Unmarshal(data, &payload)
		}
	case <-time.After(deadline):
	}
	return payload
}

func (p hookPayload) directory() string {
	for _, candidate := range []string{p.Workspace.CurrentDir, p.Cwd} {
		if candidate != "" {
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return candidate
			}
		}
	}
	working, err := os.Getwd()
	if err != nil {
		return "."
	}
	return working
}

// build assembles a view from cache only. It never runs git, because this is the
// path that executes once a second in every open session.
func build(directory, model string, settings config.Config) (render.View, bool) {
	repo, found := gitrepo.Locate(directory)
	if !found {
		return render.View{}, false
	}
	now := time.Now()
	cacheDir := config.StateDir()
	current, hasState := state.Read(cacheDir, repo.Key())
	tests := state.ReadTests(cacheDir, repo.Key(), now)

	if !hasState || current.Stale(now) {
		refreshInBackground(repo.Root)
	}

	view := render.View{
		Pet:      identity.For(repo.Branch),
		Branch:   repo.Branch,
		Root:     repo.Root,
		State:    current,
		Tests:    tests,
		Model:    model,
		HasState: hasState,
	}
	if hasState {
		view.Score = score.Of(current, tests, settings)
	}
	return view, true
}

// refreshInBackground detaches a probe so the render never waits on git.
func refreshInBackground(root string) {
	executable, err := os.Executable()
	if err != nil {
		return
	}
	command := exec.Command(executable, "probe", root)
	command.Stdout, command.Stderr = nil, nil
	if command.Start() == nil {
		go command.Wait()
	}
}

func flagValue(args []string, name, fallback string) string {
	for _, arg := range args {
		if value, found := stringsCutPrefix(arg, "--"+name+"="); found {
			return value
		}
	}
	return fallback
}

func stringsCutPrefix(text, prefix string) (string, bool) {
	if len(text) >= len(prefix) && text[:len(prefix)] == prefix {
		return text[len(prefix):], true
	}
	return "", false
}

func renderCommand(args []string) {
	settings := config.Load()
	payload := readPayload()
	// A surface with no payload can name the worktree itself. tmux passes
	// #{pane_current_path} this way.
	directory := flagValue(args, "cwd", "")
	if directory == "" {
		directory = payload.directory()
	}
	view, found := build(directory, payload.Model.DisplayName, settings)
	format := flagValue(args, "format", "statusline")
	if !found {
		if format == "json" {
			fmt.Println(`{"ready":false}`)
		}
		return
	}
	switch format {
	case "tmux":
		fmt.Println(render.Tmux(view, settings))
	case "title":
		fmt.Println(render.Title(view, settings))
	case "json":
		encoded, err := render.JSON(view, settings)
		if err != nil {
			os.Exit(1)
		}
		fmt.Println(encoded)
	default:
		fmt.Println(render.Statusline(view, settings))
	}
}

func cardCommand(args []string) {
	settings := config.Load()
	directory := ""
	for _, arg := range args {
		if len(arg) > 0 && arg[0] != '-' {
			directory = arg
		}
	}
	if directory == "" {
		directory, _ = os.Getwd()
	}
	repo, found := gitrepo.Locate(directory)
	if !found {
		fmt.Printf("(-.-) no git worktree at %s\n", directory)
		os.Exit(1)
	}
	// The card is on demand, so it can afford to be current rather than cached.
	probe(repo.Root, settings)
	view, ok := build(repo.Root, "", settings)
	if !ok {
		os.Exit(1)
	}
	fmt.Print(render.Card(view, settings))
	fmt.Print(updateNotice(settings))
}

func probeCommand(args []string) {
	if len(args) == 0 {
		os.Exit(0)
	}
	probe(args[0], config.Load())
}

// probe holds a directory lock so several sessions refreshing at once do not
// stampede the same worktree.
func probe(root string, settings config.Config) {
	repo, found := gitrepo.Locate(root)
	if !found {
		return
	}
	cacheDir := config.StateDir()
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return
	}
	lock := filepath.Join(cacheDir, repo.Key()+".lock")
	if os.Mkdir(lock, 0o755) != nil {
		return
	}
	defer os.Remove(lock)
	state.Write(cacheDir, repo.Key(), signal.Collect(repo, settings, time.Now()))
	// A branch earns its den entry on its first commit, which may be long after
	// the hatch, so the probe is where that gets noticed.
	recordIfEarned(repo, settings, cacheDir)
}

func recordCommand() {
	payload := readPayload()
	repo, found := gitrepo.Locate(payload.directory())
	if !found {
		return
	}
	result, ok := verdict.Of(payload.ToolInput.Command, responseText(payload.ToolResponse))
	if !ok {
		return
	}
	state.WriteTests(config.StateDir(), repo.Key(), result, time.Now())
}

// responseText decodes what a runner actually printed.
//
// Casting the raw message to a string yields JSON source: quotes included and
// newlines still written as backslash-n. Substring checks survive that, but
// anything anchored to a line never matches, so the text has to be decoded.
// Harnesses send either a bare string or an object carrying stdout and stderr.
func responseText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	var fields map[string]any
	if json.Unmarshal(raw, &fields) == nil {
		var parts []string
		for _, key := range []string{"stdout", "stderr", "output", "content", "result"} {
			if value, isText := fields[key].(string); isText && value != "" {
				parts = append(parts, value)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	return string(raw)
}

func quipCommand() {
	settings := config.Load()
	payload := readPayload()
	repo, found := gitrepo.Locate(payload.directory())
	if !found {
		return
	}
	probe(repo.Root, settings)
	view, ok := build(repo.Root, "", settings)
	if !ok {
		return
	}
	message := fmt.Sprintf("%s %s: %s", view.Body(), view.Pet.Name, quipFor(view))
	encoded, err := json.Marshal(map[string]string{"systemMessage": message})
	if err != nil {
		return
	}
	fmt.Println(string(encoded))
}

// quipFor names the worst thing it can see, so the line is always the useful one.
func quipFor(view render.View) string {
	switch {
	case view.State.Migrations > 1:
		return fmt.Sprintf("%d migration heads. that one never fixes itself.", view.State.Migrations)
	case view.Tests == "fail":
		return "tests are red. i saw it."
	case view.State.Unpushed > 5:
		return fmt.Sprintf("%d unpushed. this branch only exists on your laptop.", view.State.Unpushed)
	case view.State.Dirty > 15:
		return fmt.Sprintf("%d files uncommitted. that is a lot of uncommitted.", view.State.Dirty)
	case view.State.Dirty > 0 || view.State.Unpushed > 0:
		return fmt.Sprintf("%d△ %d↑, nothing alarming.", view.State.Dirty, view.State.Unpushed)
	default:
		return "clean and pushed. rare."
	}
}
