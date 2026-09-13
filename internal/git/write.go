package git

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (c Client) StageFiles(ctx context.Context, repositoryPath string, files []string) error {
	if len(files) == 0 {
		return fmt.Errorf("at least one file must be staged")
	}
	for _, file := range files {
		if file == "" || strings.HasPrefix(file, "/") || strings.Contains(file, "..") {
			return fmt.Errorf("invalid file path: %q", file)
		}
	}

	if c.Runner == nil {
		return fmt.Errorf("git runner is not configured")
	}

	commandContext, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	arguments := append([]string{"add"}, files...)
	_, err := c.Runner.Run(commandContext, repositoryPath, arguments)
	return err
}

func (c Client) Commit(ctx context.Context, repositoryPath string, opts CommitOptions) (OperationResult, error) {
	result := OperationResult{
		Type:      "commit",
		State:     StateRunning,
		StartTime: time.Now(),
	}

	if err := opts.Validate(); err != nil {
		result.State = StateFailed
		result.Error = err.Error()
		result.EndTime = time.Now()
		return result, err
	}

	if c.Runner == nil {
		result.State = StateFailed
		result.Error = "git runner is not configured"
		result.EndTime = time.Now()
		return result, errors.New(result.Error)
	}

	// Check if repository state matches expected snapshot (stale detection)
	if opts.ExpectedSnapshotID != "" {
		currentStatus, err := c.Status(ctx, repositoryPath)
		if err != nil {
			result.State = StateFailed
			result.Error = fmt.Sprintf("failed to check repository state: %v", err)
			result.EndTime = time.Now()
			return result, errors.New(result.Error)
		}
		currentSnapshot := NewSnapshot(currentStatus)
		if currentSnapshot.ID != opts.ExpectedSnapshotID {
			result.State = StateStale
			result.Error = fmt.Sprintf("repository changed after review: expected %s, got %s", opts.ExpectedSnapshotID, currentSnapshot.ID)
			result.EndTime = time.Now()
			return result, errors.New(result.Error)
		}
	}

	commandContext, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	arguments := []string{
		"commit",
		"--no-edit",
		"-m", opts.Message,
	}
	output, err := c.Runner.Run(commandContext, repositoryPath, arguments)
	result.Output = output
	result.EndTime = time.Now()

	if err != nil {
		result.State = StateFailed
		result.Error = err.Error()
		result.Success = false
		return result, err
	}

	result.State = StateCompleted
	result.Success = true
	return result, nil
}

func (c Client) Push(ctx context.Context, repositoryPath string, opts PushOptions) (OperationResult, error) {
	result := OperationResult{
		Type:      "push",
		State:     StateRunning,
		StartTime: time.Now(),
	}

	if err := opts.Validate(); err != nil {
		result.State = StateFailed
		result.Error = err.Error()
		result.EndTime = time.Now()
		return result, err
	}

	if c.Runner == nil {
		result.State = StateFailed
		result.Error = "git runner is not configured"
		result.EndTime = time.Now()
		return result, errors.New(result.Error)
	}

	commandContext, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	arguments := []string{"push"}
	if opts.SetUpstream {
		arguments = append(arguments, "-u")
	}
	if opts.Remote != "" {
		arguments = append(arguments, opts.Remote)
		if opts.Branch != "" {
			arguments = append(arguments, opts.Branch)
		}
	} else if opts.Branch != "" {
		arguments = append(arguments, "origin", opts.Branch)
	}

	output, err := c.Runner.Run(commandContext, repositoryPath, arguments)
	result.Output = output
	result.EndTime = time.Now()

	if err != nil {
		result.State = StateFailed
		result.Error = err.Error()
		result.Success = false
		return result, err
	}

	result.State = StateCompleted
	result.Success = true
	return result, nil
}
