package signal

import (
	"os"
	"path/filepath"
	"testing"
)

func writeMigration(t *testing.T, dir, name, revision, down string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "revision = \"" + revision + "\"\n"
	if down == "" {
		body += "down_revision = None\n"
	} else {
		body += "down_revision = \"" + down + "\"\n"
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

var defaultPaths = []string{"alembic/versions", "migrations/versions"}

// The shell version globbed */alembic/versions and */*/migrations/versions, so a
// repo with alembic/versions at its root was never matched and the signal
// silently reported zero for exactly the projects it was aimed at.
func TestMigrationHeadsFindsRootLevelLayout(t *testing.T) {
	root := t.TempDir()
	versions := filepath.Join(root, "alembic", "versions")
	writeMigration(t, versions, "a.py", "aaa", "")
	writeMigration(t, versions, "b.py", "bbb", "aaa")

	if got := MigrationHeads(root, defaultPaths); got != 1 {
		t.Errorf("root-level alembic/versions gave %d heads, want 1", got)
	}
}

func TestMigrationHeadsFindsNestedLayout(t *testing.T) {
	root := t.TempDir()
	versions := filepath.Join(root, "services", "api", "migrations", "versions")
	writeMigration(t, versions, "a.py", "aaa", "")
	writeMigration(t, versions, "b.py", "bbb", "aaa")

	if got := MigrationHeads(root, defaultPaths); got != 1 {
		t.Errorf("nested migrations/versions gave %d heads, want 1", got)
	}
}

func TestMigrationHeadsDetectsASplit(t *testing.T) {
	root := t.TempDir()
	versions := filepath.Join(root, "alembic", "versions")
	writeMigration(t, versions, "a.py", "aaa", "")
	writeMigration(t, versions, "b.py", "bbb", "aaa")
	writeMigration(t, versions, "c.py", "ccc", "aaa")

	if got := MigrationHeads(root, defaultPaths); got != 2 {
		t.Errorf("two children of one parent gave %d heads, want 2", got)
	}
}

// Two trees each with one head is a healthy repo. Summing across trees would
// cry wolf, so the worst single tree wins.
func TestMigrationHeadsTakesWorstTreeNotTheSum(t *testing.T) {
	root := t.TempDir()
	for _, app := range []string{"one", "two"} {
		versions := filepath.Join(root, app, "alembic", "versions")
		writeMigration(t, versions, "a.py", app+"aaa", "")
		writeMigration(t, versions, "b.py", app+"bbb", app+"aaa")
	}
	if got := MigrationHeads(root, defaultPaths); got != 1 {
		t.Errorf("two healthy trees gave %d heads, want 1", got)
	}
}

func TestMigrationHeadsIgnoresVendorDirectories(t *testing.T) {
	root := t.TempDir()
	buried := filepath.Join(root, "node_modules", "thing", "alembic", "versions")
	writeMigration(t, buried, "a.py", "aaa", "")
	writeMigration(t, buried, "b.py", "bbb", "")

	if got := MigrationHeads(root, defaultPaths); got != 0 {
		t.Errorf("a dependency's migrations counted for %d heads, want 0", got)
	}
}

func TestMigrationHeadsOnRepoWithoutMigrations(t *testing.T) {
	if got := MigrationHeads(t.TempDir(), defaultPaths); got != 0 {
		t.Errorf("a repo with no migrations gave %d heads, want 0", got)
	}
}
