// Command pets renders a per-worktree creature and the branch hygiene behind its mood.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/TevanB/parallel-harness-pets/internal/config"
	"github.com/TevanB/parallel-harness-pets/internal/gitrepo"
	"github.com/TevanB/parallel-harness-pets/internal/identity"
	"github.com/TevanB/parallel-harness-pets/internal/render"
	"github.com/TevanB/parallel-harness-pets/internal/score"
	"github.com/TevanB/parallel-harness-pets/internal/signal"
	"github.com/TevanB/parallel-harness-pets/internal/state"
	"github.com/TevanB/parallel-harness-pets/internal/verdict"
)

var version = "dev"

const usage = `pets - a creature for every worktree

  pets render [--format=statusline|tmux|title|json]  render for a surface
  pets card [path]                                   full readout
  pets probe <path>                                  refresh one worktree's cache
  pets record                                        tool-use hook, reads JSON on stdin
  pets quip                                          stop hook, reads JSON on stdin
  pets install [--harness=claude|codex|tmux|shell|all]
  pets uninstall
  pets version
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		return
	}
	switch os.Args[1] {
	case "render":
		renderCommand(os.Args[2:])
	case "card":
		cardCommand(os.Args[2:])
	case "probe":
		probeCommand(os.Args[2:])
	case "record":
		recordCommand()
	case "quip":
		quipCommand()
	case "install", "uninstall":
		installCommand(os.Args[1], os.Args[2:])
	case "version", "--version", "-v":
		fmt.Println(version)
	default:
		fmt.Print(usage)
		os.Exit(1)
	}
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

// readPayload parses harness JSON from stdin, tolerating an empty or absent one.
func readPayload() hookPayload {
	var payload hookPayload
	info, err := os.Stdin.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice != 0 {
		return payload
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil || len(data) == 0 {
		return payload
	}
	json.Unmarshal(data, &payload)
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
	view, found := build(payload.directory(), payload.Model.DisplayName, settings)
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
}

func recordCommand() {
	payload := readPayload()
	repo, found := gitrepo.Locate(payload.directory())
	if !found {
		return
	}
	result, ok := verdict.Of(payload.ToolInput.Command, string(payload.ToolResponse))
	if !ok {
		return
	}
	state.WriteTests(config.StateDir(), repo.Key(), result, time.Now())
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
