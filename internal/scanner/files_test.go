package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadSelectedFileAllowsRepositoryRelativeFile(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "notes"), 0o755); err != nil {
		t.Fatalf("create notes directory: %v", err)
	}
	createFile(t, filepath.Join(root, "notes", "draft.txt"), "draft contents")

	contents, err := ReadSelectedFile(root, "notes/draft.txt", 100)
	if err != nil {
		t.Fatalf("ReadSelectedFile returned an error: %v", err)
	}
	if string(contents) != "draft contents" {
		t.Fatalf("contents = %q", contents)
	}
}

func TestReadSelectedFileRejectsEscapeAndOversize(t *testing.T) {
	root := t.TempDir()
	createFile(t, filepath.Join(root, "small.txt"), "12345")

	if _, err := ReadSelectedFile(root, "../outside.txt", 100); err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("escape error = %v", err)
	}
	if _, err := ReadSelectedFile(root, "small.txt", 3); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("size error = %v", err)
	}
}

func TestReadSelectedFileRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	createFile(t, filepath.Join(outside, "secret.txt"), "secret")
	link := filepath.Join(root, "secret.txt")
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), link); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}

	if _, err := ReadSelectedFile(root, "secret.txt", 100); err == nil || !strings.Contains(err.Error(), "symlink escapes") {
		t.Fatalf("symlink error = %v", err)
	}
}
