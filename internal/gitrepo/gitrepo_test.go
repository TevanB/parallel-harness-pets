package gitrepo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// Relative paths have to be resolved against the current directory before the
// walk starts, otherwise a path like "../b" from inside "repo/a" stops at the
// current directory and misses the repository one level up.
func TestLocateWalksUpFromRelativePath(t *testing.T) {
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	os.MkdirAll(gitDir, 0o755)
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0o644)
	nested := filepath.Join(root, "a", "b")
	os.MkdirAll(nested, 0o755)

	t.Chdir(filepath.Join(root, "a"))
	repo, found := Locate("../b")
	if !found || repo.Root != root {
		t.Fatalf("Locate(../b) from %s = %+v/%v, want root %q", filepath.Join(root, "a"), repo, found, root)
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

// A cache key becomes a filename, so it must survive every platform's rules.
// Replacing only the path separator left the Windows drive colon in place, and
// a colon cannot appear in a Windows filename at all.
func TestKeyIsFilenameSafeOnEveryPlatform(t *testing.T) {
	roots := []string{
		"/Users/x/code/my-repo",
		`C:\Users\x\code\my-repo`,
		"/tmp/repo with spaces/and:colons",
		"/",
	}
	illegal := `<>:"/\|?*`
	for _, root := range roots {
		key := Repo{Root: root}.Key()
		if key == "" {
			t.Errorf("Key() for %q is empty", root)
		}
		if strings.ContainsAny(key, illegal) {
			t.Errorf("Key() for %q = %q, which contains an illegal filename character", root, key)
		}
		if filepath.Base(key) != key {
			t.Errorf("Key() for %q = %q, which is not a single path segment", root, key)
		}
		if len(key) > 128 {
			t.Errorf("Key() for %q is %d bytes, too long for some filesystems", root, len(key))
		}
	}
}

// Two paths that sanitise to the same text must not share a cache entry.
func TestKeyDistinguishesPathsThatSanitiseAlike(t *testing.T) {
	first := Repo{Root: "/a/b-c"}.Key()
	second := Repo{Root: "/a_b/c"}.Key()
	if first == second {
		t.Errorf("distinct roots collapsed onto one key: %q", first)
	}
}

func TestKeyIsStable(t *testing.T) {
	root := "/Users/x/code/my-repo"
	if (Repo{Root: root}).Key() != (Repo{Root: root}).Key() {
		t.Error("Key() is not stable, so the cache would be rewritten every call")
	}
}

// An explicit default branch must win, so anyone on a non-standard trunk can fix it.
func TestDefaultBranchHonoursOverride(t *testing.T) {
	if got := DefaultBranch(t.TempDir(), "origin/trunk"); got != "origin/trunk" {
		t.Errorf("DefaultBranch with override = %q, want origin/trunk", got)
	}
}

// A repo with no remote still has a trunk. Probing only remote refs left every
// comparison failing silently: behind and unpushed read zero, and no branch
// could ever earn its place in the den.
func TestDefaultBranchFindsALocalTrunk(t *testing.T) {
	root := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "."},
		{"config", "user.email", "a@b.c"},
		{"config", "user.name", "A"},
	} {
		if err := exec.Command("git", append([]string{"-C", root}, args...)...).Run(); err != nil {
			t.Skipf("git unavailable: %v", err)
		}
	}
	os.WriteFile(filepath.Join(root, "a.txt"), []byte("hi"), 0o644)
	exec.Command("git", "-C", root, "add", "-A").Run()
	exec.Command("git", "-C", root, "commit", "-qm", "init").Run()
	exec.Command("git", "-C", root, "branch", "-M", "main").Run()

	if got := DefaultBranch(root, ""); got != "main" {
		t.Errorf("DefaultBranch on a remoteless repo = %q, want the local main", got)
	}
}

func TestIsTrunkAcceptsLocalAndRemoteNames(t *testing.T) {
	cases := []struct {
		trunk, branch string
		want          bool
	}{
		{"origin/main", "main", true},
		{"main", "main", true},
		{"origin/master", "master", true},
		{"master", "master", true},
		{"origin/main", "feat/thing", false},
		{"main", "feat/thing", false},
		// A branch merely ending in the trunk's name is not the trunk.
		{"origin/main", "not-main", false},
	}
	for _, testCase := range cases {
		if got := IsTrunk(testCase.trunk, testCase.branch); got != testCase.want {
			t.Errorf("IsTrunk(%q, %q) = %v, want %v",
				testCase.trunk, testCase.branch, got, testCase.want)
		}
	}
}
