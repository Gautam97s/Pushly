package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanFindsNestedRepositoriesAndWorktrees(t *testing.T) {
	root := t.TempDir()
	createDirectory(t, filepath.Join(root, "Projects", "App", ".git"))
	createDirectory(t, filepath.Join(root, "Projects", "Nested"))
	createFile(t, filepath.Join(root, "Projects", "Nested", ".git"), "gitdir: C:/worktrees/app")
	createDirectory(t, filepath.Join(root, "Notes"))

	result, err := Scan(root, Options{MaxDepth: 5, MaxRepositories: 10})
	if err != nil {
		t.Fatalf("Scan returned an error: %v", err)
	}
	if len(result.Repositories) != 2 {
		t.Fatalf("repositories = %d, want 2: %+v", len(result.Repositories), result.Repositories)
	}
}

func TestScanHonorsDepthAndRepositoryLimit(t *testing.T) {
	root := t.TempDir()
	createDirectory(t, filepath.Join(root, "one", ".git"))
	createDirectory(t, filepath.Join(root, "two", ".git"))
	createDirectory(t, filepath.Join(root, "deep", "repo", ".git"))

	result, err := Scan(root, Options{MaxDepth: 1, MaxRepositories: 1})
	if err != nil {
		t.Fatalf("Scan returned an error: %v", err)
	}
	if len(result.Repositories) != 1 {
		t.Fatalf("repositories = %d, want 1", len(result.Repositories))
	}
}

func TestIsWithinRejectsOutsidePaths(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "Projects")
	inside := filepath.Join(parent, "App")
	outside := filepath.Join(filepath.Dir(parent), "Other")

	within, err := IsWithin(parent, inside)
	if err != nil || !within {
		t.Fatalf("inside path result = %v, %v", within, err)
	}
	within, err = IsWithin(parent, outside)
	if err != nil || within {
		t.Fatalf("outside path result = %v, %v", within, err)
	}
}

func TestScanSkipsSymlinks(t *testing.T) {
	root := t.TempDir()
	realRepository := filepath.Join(root, "real", ".git")
	createDirectory(t, realRepository)
	linkPath := filepath.Join(root, "linked")
	if err := os.Symlink(filepath.Dir(realRepository), linkPath); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}

	result, err := Scan(root, Options{MaxDepth: 4, MaxRepositories: 10})
	if err != nil {
		t.Fatalf("Scan returned an error: %v", err)
	}
	if len(result.Repositories) != 1 {
		t.Fatalf("repositories = %d, want 1", len(result.Repositories))
	}
	if len(result.Skipped) != 1 {
		t.Fatalf("skipped = %d, want 1", len(result.Skipped))
	}
}

func createDirectory(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("create directory: %v", err)
	}
}

func createFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("create file: %v", err)
	}
}
