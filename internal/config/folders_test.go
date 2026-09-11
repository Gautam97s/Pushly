package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFolderStoreNormalizesAndRejectsOverlaps(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "Projects")
	second := filepath.Join(root, "Other")
	if err := os.MkdirAll(filepath.Join(first, "App"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(second, 0o755); err != nil {
		t.Fatal(err)
	}

	var store FolderStore
	if err := store.AddFolder(first); err != nil {
		t.Fatalf("AddFolder returned an error: %v", err)
	}
	if err := store.AddFolder(filepath.Join(first, ".")); err == nil || !strings.Contains(err.Error(), "overlaps") {
		t.Fatalf("duplicate folder error = %v", err)
	}
	if err := store.AddFolder(filepath.Join(first, "App")); err == nil || !strings.Contains(err.Error(), "overlaps") {
		t.Fatalf("nested folder error = %v", err)
	}
	if err := store.AddFolder(second); err != nil {
		t.Fatalf("AddFolder returned an error: %v", err)
	}
	if len(store.Folders) != 2 {
		t.Fatalf("folders = %d, want 2", len(store.Folders))
	}
}

func TestFolderStoreRejectsMissingAndNonDirectoryPaths(t *testing.T) {
	var store FolderStore
	if err := store.AddFolder(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing folder was accepted")
	}
	file := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := store.AddFolder(file); err == nil {
		t.Fatal("file path was accepted as a folder")
	}
}
