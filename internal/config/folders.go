package config

import (
	"fmt"
	"os"
	"path/filepath"

	"pushly/internal/scanner"
)

type FolderStore struct {
	Folders []string `json:"folders"`
}

func (store *FolderStore) AddFolder(path string) error {
	normalized, err := normalizeDirectory(path)
	if err != nil {
		return err
	}

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
	for index, existing := range store.Folders {
		if existing == normalized {
			store.Folders = append(store.Folders[:index], store.Folders[index+1:]...)
			return nil
		}
	}
	return fmt.Errorf("folder is not configured: %s", normalized)
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
