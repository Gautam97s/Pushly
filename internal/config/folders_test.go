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

func TestFolderStorePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "Pushly", "config.json")

	// Loading non-existent file should return empty store without error
	store, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if len(store.ListFolders()) != 0 {
		t.Fatalf("expected 0 folders, got %d", len(store.ListFolders()))
	}

	folder1 := filepath.Join(tmpDir, "Projects")
	if err := os.MkdirAll(folder1, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := store.AddFolder(folder1); err != nil {
		t.Fatalf("AddFolder error: %v", err)
	}

	if err := store.Save(configPath); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Reload config from disk
	loadedStore, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig after save error: %v", err)
	}

	folders := loadedStore.ListFolders()
	if len(folders) != 1 || folders[0] != folder1 {
		t.Fatalf("loaded folders mismatch: %v", folders)
	}

	// Test IsApproved
	approved, err := loadedStore.IsApproved(filepath.Join(folder1, "my-repo"))
	if err != nil || !approved {
		t.Fatalf("IsApproved failed: approved=%v, err=%v", approved, err)
	}

	unapproved, err := loadedStore.IsApproved(filepath.Join(tmpDir, "unapproved-repo"))
	if err != nil || unapproved {
		t.Fatalf("IsApproved should be false for unapproved repo: approved=%v, err=%v", unapproved, err)
	}

	// Test RemoveFolder
	if err := loadedStore.RemoveFolder(folder1); err != nil {
		t.Fatalf("RemoveFolder error: %v", err)
	}
	if len(loadedStore.ListFolders()) != 0 {
		t.Fatalf("expected 0 folders after removal, got %d", len(loadedStore.ListFolders()))
	}
	if err := loadedStore.RemoveFolder(folder1); err == nil {
		t.Fatal("expected error when removing non-configured folder")
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath returned error: %v", err)
	}
	if path == "" {
		t.Fatal("DefaultConfigPath returned empty string")
	}
	if !strings.HasSuffix(path, "config.json") {
		t.Fatalf("DefaultConfigPath expected to end with config.json, got: %s", path)
	}
}
