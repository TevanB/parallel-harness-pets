// Package gitrepo locates a worktree and reads its branch without running git.
//
// The status line renders once a second in every open session, so the hot path
// reads two small files instead of paying for a subprocess.
package gitrepo

import (
	"fmt"
	"hash/fnv"
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
	if !filepath.IsAbs(dir) {
		if abs, err := filepath.Abs(dir); err == nil {
			dir = abs
		}
	}
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

// Key turns a worktree path into a filename safe on every platform.
//
// Replacing only the path separator is not enough: on Windows that leaves the
// drive colon in place, and `< > : " / \ | ? *` are all illegal in a filename,
// so the cache file could never be created. Anything outside the safe set is
// folded to an underscore, and an FNV suffix keeps two paths that sanitise
// alike from sharing one cache entry.
func (r Repo) Key() string {
	var flat strings.Builder
	for _, char := range r.Root {
		switch {
		case char >= 'a' && char <= 'z', char >= 'A' && char <= 'Z',
			char >= '0' && char <= '9', char == '-', char == '.':
			flat.WriteRune(char)
		default:
			flat.WriteRune('_')
		}
	}
	trimmed := strings.Trim(flat.String(), "_")
	// Keep the tail, which is the part a human recognises, and stay well inside
	// the 255-byte limit most filesystems impose.
	if len(trimmed) > 80 {
		trimmed = trimmed[len(trimmed)-80:]
	}
	digest := fnv.New32a()
	digest.Write([]byte(r.Root))
	return fmt.Sprintf("%s-%08x", trimmed, digest.Sum32())
}

// Worktree is the name git itself uses for this worktree, matching what a
// harness reports as workspace.git_worktree. A linked worktree keeps its git
// directory at <main>/.git/worktrees/<name>, so the name is already on disk.
func (r Repo) Worktree() string {
	if parent := filepath.Dir(r.GitDir); filepath.Base(parent) == "worktrees" {
		return filepath.Base(r.GitDir)
	}
	return filepath.Base(r.Root)
}

// Project is the repository every one of its worktrees shares, so sibling
// worktrees of one repo cannot be mistaken for worktrees of another.
func (r Repo) Project() string {
	if parent := filepath.Dir(r.GitDir); filepath.Base(parent) == "worktrees" {
		// <main>/.git/worktrees/<name> -> <main>
		return filepath.Base(filepath.Dir(filepath.Dir(parent)))
	}
	return filepath.Base(r.Root)
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
	// Remote refs first, then local ones. A repo with no remote still has a
	// trunk, and without this fallback every comparison against it fails
	// silently: behind and unpushed read zero, and no branch can ever earn its
	// place in the den.
	candidates := []string{
		"origin/main", "origin/master", "origin/trunk", "origin/develop",
		"main", "master", "trunk", "develop",
	}
	for _, candidate := range candidates {
		if exec.Command("git", "-C", root, "rev-parse", "--verify", "--quiet", candidate).Run() == nil {
			return candidate
		}
	}
	return "origin/main"
}

// IsTrunk reports whether a branch is the repo's trunk, accepting both a local
// name and a remote-qualified one.
func IsTrunk(trunk, branch string) bool {
	return trunk == branch || strings.HasSuffix(trunk, "/"+branch)
}
