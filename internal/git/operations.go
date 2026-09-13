package git

import (
	"fmt"
	"strings"
	"time"
)

type FileSelection struct {
	RepositoryRoot string
	Paths          []string
}

func (selection *FileSelection) Validate() error {
	if selection.RepositoryRoot == "" {
		return fmt.Errorf("repository root is required")
	}
	if len(selection.Paths) == 0 {
		return fmt.Errorf("at least one file must be selected")
	}

	for _, path := range selection.Paths {
		if path == "" || strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") {
			return fmt.Errorf("invalid file path: %q", path)
		}
		if strings.Contains(path, "..") {
			return fmt.Errorf("file path cannot contain traversal: %q", path)
		}
	}
	return nil
}

type OperationState string

const (
	StatePending   OperationState = "pending"
	StateRunning   OperationState = "running"
	StateCompleted OperationState = "completed"
	StateFailed    OperationState = "failed"
	StateStale     OperationState = "stale"
)

type OperationResult struct {
	ID        string
	Type      string
	State     OperationState
	StartTime time.Time
	EndTime   time.Time
	Output    string
	Error     string
	Success   bool
}

func (result *OperationResult) Duration() time.Duration {
	if result.EndTime.IsZero() || result.StartTime.IsZero() {
		return 0
	}
	return result.EndTime.Sub(result.StartTime)
}

type CommitOptions struct {
	SelectedFiles      []string
	Message            string
	ExpectedSnapshotID string
}

func (opts *CommitOptions) Validate() error {
	if len(opts.SelectedFiles) == 0 {
		return fmt.Errorf("commit requires at least one selected file")
	}
	if strings.TrimSpace(opts.Message) == "" {
		return fmt.Errorf("commit message cannot be empty")
	}
	if len(opts.Message) > 1000 {
		return fmt.Errorf("commit message exceeds 1000 characters")
	}
	for _, file := range opts.SelectedFiles {
		if file == "" || strings.HasPrefix(file, "/") || strings.Contains(file, "..") {
			return fmt.Errorf("invalid file in commit: %q", file)
		}
	}
	return nil
}

type PushOptions struct {
	Remote      string
	Branch      string
	SetUpstream bool
}

func (opts *PushOptions) Validate() error {
	if opts.Remote != "" {
		if strings.ContainsAny(opts.Remote, " \t\n\r;`$&|><") || strings.HasPrefix(opts.Remote, "-") {
			return fmt.Errorf("invalid remote name: %q", opts.Remote)
		}
	}
	if opts.Branch != "" {
		if strings.ContainsAny(opts.Branch, " \t\n\r;`$&|><~^:?*[\\") || strings.HasPrefix(opts.Branch, "-") {
			return fmt.Errorf("invalid branch name: %q", opts.Branch)
		}
	}
	return nil
}
