package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Options struct {
	MaxDepth        int
	MaxRepositories int
}

type Repository struct {
	Name string
	Path string
}

type SkippedPath struct {
	Path   string
	Reason string
}

type Result struct {
	Repositories []Repository
	Skipped      []SkippedPath
}

func Scan(root string, options Options) (Result, error) {
	rootPath, err := normalizePath(root)
	if err != nil {
		return Result{}, err
	}

	rootInfo, err := os.Lstat(rootPath)
	if err != nil {
		return Result{}, fmt.Errorf("inspect scan root: %w", err)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 {
		return Result{}, fmt.Errorf("scan root cannot be a symlink: %s", rootPath)
	}
	if !rootInfo.IsDir() {
		return Result{}, fmt.Errorf("scan root is not a directory: %s", rootPath)
	}

	result := Result{}
	visited := make(map[string]bool)
	var scanDirectory func(string, int)
	scanDirectory = func(directory string, depth int) {
		if options.MaxRepositories > 0 && len(result.Repositories) >= options.MaxRepositories {
			return
		}
		key := strings.ToLower(directory)
		if visited[key] {
			return
		}
		visited[key] = true

		if isRepository(directory) {
			result.Repositories = append(result.Repositories, Repository{
				Name: filepath.Base(directory),
				Path: directory,
			})
		}
		if options.MaxDepth >= 0 && depth >= options.MaxDepth {
			return
		}

		entries, err := os.ReadDir(directory)
		if err != nil {
			result.Skipped = append(result.Skipped, SkippedPath{Path: directory, Reason: err.Error()})
			return
		}
		for _, entry := range entries {
			if options.MaxRepositories > 0 && len(result.Repositories) >= options.MaxRepositories {
				return
			}
			entryPath := filepath.Join(directory, entry.Name())
			if entry.Type()&os.ModeSymlink != 0 {
				result.Skipped = append(result.Skipped, SkippedPath{Path: entryPath, Reason: "symlink skipped"})
				continue
			}
			if !entry.IsDir() || entry.Name() == ".git" {
				continue
			}
			scanDirectory(entryPath, depth+1)
		}
	}

	scanDirectory(rootPath, 0)
	return result, nil
}

func IsWithin(parent, candidate string) (bool, error) {
	parentPath, err := normalizePath(parent)
	if err != nil {
		return false, err
	}
	candidatePath, err := normalizePath(candidate)
	if err != nil {
		return false, err
	}
	relative, err := filepath.Rel(parentPath, candidatePath)
	if err != nil {
		return false, err
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)), nil
}

func normalizePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absPath), nil
}

func isRepository(directory string) bool {
	_, err := os.Lstat(filepath.Join(directory, ".git"))
	return err == nil
}
