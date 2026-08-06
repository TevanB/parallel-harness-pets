package gitrepo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocatePlainRepo(t *testing.T) {
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/feat/checkout-flow\n"), 0o644)

	repo, found := Locate(filepath.Join(root))
	if !found {
		t.Fatal("Locate did not find the repo")
	}
	if repo.Branch != "feat/checkout-flow" {
		t.Errorf("branch = %q, want feat/checkout-flow", repo.Branch)
	}
}

func TestLocateWalksUpFromASubdirectory(t *testing.T) {
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	os.MkdirAll(gitDir, 0o755)
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0o644)
	nested := filepath.Join(root, "a", "b", "c")
	os.MkdirAll(nested, 0o755)

	repo, found := Locate(nested)
	if !found || repo.Root != root {
		t.Fatalf("Locate(%q) = %+v/%v, want root %q", nested, repo, found, root)
	}
}

// A linked worktree has a .git file pointing elsewhere, not a .git directory.
// This is the whole point of the project, so it had better work.
func TestLocateLinkedWorktree(t *testing.T) {
	base := t.TempDir()
	realGitDir := filepath.Join(base, "main", ".git", "worktrees", "feature")
	os.MkdirAll(realGitDir, 0o755)
	os.WriteFile(filepath.Join(realGitDir, "HEAD"), []byte("ref: refs/heads/spike/wasm-build\n"), 0o644)

	tree := filepath.Join(base, "tree")
	os.MkdirAll(tree, 0o755)
	os.WriteFile(filepath.Join(tree, ".git"), []byte("gitdir: "+realGitDir+"\n"), 0o644)

	repo, found := Locate(tree)
	if !found {
		t.Fatal("Locate did not follow the gitdir pointer")
	}
	if repo.Branch != "spike/wasm-build" {
		t.Errorf("branch = %q, want spike/wasm-build", repo.Branch)
	}
	if repo.Root != tree {
		t.Errorf("root = %q, want the worktree %q", repo.Root, tree)
	}
}

func TestLocateDetachedHead(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".git"), 0o755)
	os.WriteFile(filepath.Join(root, ".git", "HEAD"), []byte("a1b2c3d4e5f6a7b8c9\n"), 0o644)

	repo, found := Locate(root)
	if !found || repo.Branch != "a1b2c3d" {
		t.Errorf("detached HEAD gave %q, want the short sha a1b2c3d", repo.Branch)
	}
}

func TestLocateOutsideAnyRepo(t *testing.T) {
	if _, found := Locate(t.TempDir()); found {
		t.Error("Locate found a repo where there is none")
	}
}

func TestKeyIsFilenameSafe(t *testing.T) {
	key := Repo{Root: "/Users/x/code/my-repo"}.Key()
	if filepath.Base(key) != key {
		t.Errorf("Key() = %q, which is not a single path segment", key)
	}
}

// An explicit default branch must win, so anyone on a non-standard trunk can fix it.
func TestDefaultBranchHonoursOverride(t *testing.T) {
	if got := DefaultBranch(t.TempDir(), "origin/trunk"); got != "origin/trunk" {
		t.Errorf("DefaultBranch with override = %q, want origin/trunk", got)
	}
}
