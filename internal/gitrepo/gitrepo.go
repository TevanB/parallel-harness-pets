// Package gitrepo locates a worktree and reads its branch without running git.
//
// The status line renders once a second in every open session, so the hot path
// reads two small files instead of paying for a subprocess.
package gitrepo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Repo struct {
	Root   string
	GitDir string
	Branch string
}

// Locate walks up from dir looking for .git, honouring the gitdir: pointer file
// that a linked worktree uses in place of a directory.
func Locate(dir string) (Repo, bool) {
	for dir != "" && dir != string(filepath.Separator) {
		info, err := os.Stat(filepath.Join(dir, ".git"))
		switch {
		case err == nil && info.IsDir():
			return read(dir, filepath.Join(dir, ".git"))
		case err == nil:
			pointer, err := os.ReadFile(filepath.Join(dir, ".git"))
			if err != nil {
				return Repo{}, false
			}
			target := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(pointer)), "gitdir:"))
			if !filepath.IsAbs(target) {
				target = filepath.Join(dir, target)
			}
			return read(dir, target)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return Repo{}, false
}

func read(root, gitDir string) (Repo, bool) {
	head, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		return Repo{}, false
	}
	line := strings.TrimSpace(string(head))
	branch := line
	if reference, found := strings.CutPrefix(line, "ref: "); found {
		branch = strings.TrimPrefix(reference, "refs/heads/")
	} else if len(line) > 7 {
		branch = line[:7]
	}
	if branch == "" {
		return Repo{}, false
	}
	return Repo{Root: root, GitDir: gitDir, Branch: branch}, true
}

// Key turns a worktree path into a filename safe for the cache directory.
func (r Repo) Key() string {
	return strings.ReplaceAll(strings.TrimPrefix(r.Root, string(filepath.Separator)), string(filepath.Separator), "_")
}

// DefaultBranch resolves what this repo calls its trunk, so master and trunk
// repos are not silently compared against a main that does not exist.
func DefaultBranch(root, override string) string {
	if override != "" {
		return override
	}
	out, err := exec.Command("git", "-C", root, "symbolic-ref", "--short", "refs/remotes/origin/HEAD").Output()
	if err == nil {
		if name := strings.TrimSpace(string(out)); name != "" {
			return name
		}
	}
	for _, candidate := range []string{"origin/main", "origin/master", "origin/trunk", "origin/develop"} {
		if exec.Command("git", "-C", root, "rev-parse", "--verify", "--quiet", candidate).Run() == nil {
			return candidate
		}
	}
	return "origin/main"
}
