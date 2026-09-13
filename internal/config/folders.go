package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"pushly/internal/scanner"
)

type FolderStore struct {
	mu      sync.RWMutex
	Folders []string `json:"folders"`
}

func DefaultConfigPath() (string, error) {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData != "" {
			return filepath.Join(appData, "Pushly", "config.json"), nil
		}
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(configDir, "pushly", "config.json"), nil
}

func LoadConfig(path string) (*FolderStore, error) {
	store := &FolderStore{
		Folders: make([]string, 0),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return store, nil
		}
		return nil, fmt.Errorf("read config file %q: %w", path, err)
	}

	if len(data) == 0 {
		return store, nil
	}

	if err := json.Unmarshal(data, store); err != nil {
		return nil, fmt.Errorf("parse config file %q: %w", path, err)
	}

	if store.Folders == nil {
		store.Folders = make([]string, 0)
	}

	return store, nil
}

func (store *FolderStore) Save(path string) error {
	store.mu.RLock()
	data, err := json.MarshalIndent(store, "", "  ")
	store.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory %q: %w", dir, err)
	}

	tempFile, err := os.CreateTemp(dir, "config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp config file: %w", err)
	}
	tempName := tempFile.Name()
	defer os.Remove(tempName)

	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return fmt.Errorf("write temp config file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temp config file: %w", err)
	}

	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("atomic rename to config file %q: %w", path, err)
	}

	return nil
}

func (store *FolderStore) ListFolders() []string {
	store.mu.RLock()
	defer store.mu.RUnlock()

	result := make([]string, len(store.Folders))
	copy(result, store.Folders)
	return result
}

func (store *FolderStore) AddFolder(path string) error {
	normalized, err := normalizeDirectory(path)
	if err != nil {
		return err
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	for _, existing := range store.Folders {
		withinExisting, err := scanner.IsWithin(existing, normalized)
		if err != nil {
			return err
		}
		withinNew, err := scanner.IsWithin(normalized, existing)
		if err != nil {
			return err
		}
		if withinExisting || withinNew {
			return fmt.Errorf("folder overlaps configured folder: %s", existing)
		}
	}

	store.Folders = append(store.Folders, normalized)
	return nil
}

func (store *FolderStore) RemoveFolder(path string) error {
	normalized, err := normalizeDirectory(path)
	if err != nil {
		return err
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	for index, existing := range store.Folders {
		if existing == normalized {
			store.Folders = append(store.Folders[:index], store.Folders[index+1:]...)
			return nil
		}
	}
	return fmt.Errorf("folder is not configured: %s", normalized)
}

func (store *FolderStore) IsApproved(repoPath string) (bool, error) {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return false, fmt.Errorf("normalize repository path: %w", err)
	}
	normalized := filepath.Clean(absPath)

	store.mu.RLock()
	defer store.mu.RUnlock()

	for _, folder := range store.Folders {
		within, err := scanner.IsWithin(folder, normalized)
		if err != nil {
			return false, err
		}
		if within {
			return true, nil
		}
	}
	return false, nil
}

func normalizeDirectory(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("normalize folder: %w", err)
	}
	normalized := filepath.Clean(absPath)
	info, err := os.Stat(normalized)
	if err != nil {
		return "", fmt.Errorf("inspect folder: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("configured path is not a directory: %s", normalized)
	}
	return normalized, nil
}

