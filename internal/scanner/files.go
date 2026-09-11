package scanner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func ReadSelectedFile(repositoryRoot, relativePath string, maxBytes int64) ([]byte, error) {
	if relativePath == "" || filepath.IsAbs(relativePath) {
		return nil, fmt.Errorf("file path must be repository-relative")
	}
	if maxBytes <= 0 {
		return nil, fmt.Errorf("maximum file size must be positive")
	}

	root, err := normalizePath(repositoryRoot)
	if err != nil {
		return nil, err
	}
	candidate := filepath.Join(root, filepath.FromSlash(relativePath))
	within, err := IsWithin(root, candidate)
	if err != nil {
		return nil, err
	}
	if !within {
		return nil, fmt.Errorf("file path escapes repository root")
	}

	evaluatedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve repository root: %w", err)
	}
	evaluatedCandidate, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return nil, fmt.Errorf("resolve selected file: %w", err)
	}
	within, err = IsWithin(evaluatedRoot, evaluatedCandidate)
	if err != nil {
		return nil, err
	}
	if !within {
		return nil, fmt.Errorf("selected file symlink escapes repository root")
	}

	info, err := os.Stat(evaluatedCandidate)
	if err != nil {
		return nil, fmt.Errorf("inspect selected file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("selected path is not a regular file")
	}

	file, err := os.Open(evaluatedCandidate)
	if err != nil {
		return nil, fmt.Errorf("open selected file: %w", err)
	}
	defer file.Close()

	limited := io.LimitReader(file, maxBytes+1)
	contents, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read selected file: %w", err)
	}
	if int64(len(contents)) > maxBytes {
		return nil, fmt.Errorf("selected file exceeds %d-byte limit", maxBytes)
	}
	return contents, nil
}
