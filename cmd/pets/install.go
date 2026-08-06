package main

import (
	"fmt"
	"os"

	"github.com/TevanB/parallel-harness-pets/internal/harness"
)

func installCommand(action string, args []string) {
	remove := action == "uninstall"
	binary, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot resolve own path:", err)
		os.Exit(1)
	}

	requested := flagValue(args, "harness", "")
	var targets []harness.Name
	switch requested {
	case "", "all":
		targets = harness.Detect()
		if len(targets) == 0 && !remove {
			fmt.Println("No Claude Code or Codex install found. The tmux and shell")
			fmt.Println("surfaces below work with any agent and need neither.")
		}
	default:
		targets = []harness.Name{harness.Name(requested)}
	}

	for _, target := range targets {
		var err error
		switch target {
		case harness.Claude:
			err = harness.InstallClaude(binary, remove)
		case harness.Codex:
			err = harness.InstallCodex(binary, remove)
		case harness.Tmux, harness.Shell:
			continue
		default:
			err = fmt.Errorf("unknown harness %q", target)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", target, err)
			os.Exit(1)
		}
		fmt.Printf("%s %s\n", map[bool]string{true: "removed from", false: "installed into"}[remove], target)
	}

	if remove {
		fmt.Println("\nIf you added the tmux or shell snippets by hand, remove those too.")
		return
	}
	fmt.Println("\nStart a new session to see it. Claude Code reads hooks at startup.")
	fmt.Println("\nFor tmux, or any agent without hook support:")
	fmt.Print("\n" + harness.TmuxSnippet(binary))
	fmt.Print("\n" + harness.ShellSnippet(binary))
}
