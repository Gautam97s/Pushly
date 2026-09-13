package agent

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"pushly/internal/config"
	"pushly/internal/git"
	"pushly/internal/scanner"
)

type Agent struct {
	ConfigPath string
	Store      *config.FolderStore
	GitClient  git.Client
	Queue      *OperationQueue
}

func NewAgent(configPath string, gitTimeout time.Duration) (*Agent, error) {
	if configPath == "" {
		defaultPath, err := config.DefaultConfigPath()
		if err != nil {
			return nil, fmt.Errorf("resolve default config path: %w", err)
		}
		configPath = defaultPath
	}

	store, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	return &Agent{
		ConfigPath: configPath,
		Store:      store,
		GitClient:  git.NewClient(gitTimeout),
		Queue:      NewOperationQueue(),
	}, nil
}

func NewAgentWithStore(store *config.FolderStore, client git.Client, queue *OperationQueue) *Agent {
	if queue == nil {
		queue = NewOperationQueue()
	}
	return &Agent{
		Store:     store,
		GitClient: client,
		Queue:     queue,
	}
}

func (a *Agent) SaveConfig() error {
	if a.ConfigPath == "" {
		return nil
	}
	return a.Store.Save(a.ConfigPath)
}

func (a *Agent) AddFolder(path string) error {
	if err := a.Store.AddFolder(path); err != nil {
		return err
	}
	return a.SaveConfig()
}

func (a *Agent) RemoveFolder(path string) error {
	if err := a.Store.RemoveFolder(path); err != nil {
		return err
	}
	return a.SaveConfig()
}

func (a *Agent) ListFolders() []string {
	return a.Store.ListFolders()
}

func (a *Agent) ValidateRepository(ctx context.Context, repoPath string) (string, error) {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	cleanPath := filepath.Clean(absPath)

	approved, err := a.Store.IsApproved(cleanPath)
	if err != nil {
		return "", fmt.Errorf("check approved folders: %w", err)
	}
	if !approved {
		return "", fmt.Errorf("repository %q is not within any approved folder", cleanPath)
	}

	root, err := a.GitClient.RepositoryRoot(ctx, cleanPath)
	if err != nil {
		return "", fmt.Errorf("validate git repository: %w", err)
	}

	rootApproved, err := a.Store.IsApproved(root)
	if err != nil || !rootApproved {
		return "", fmt.Errorf("git repository root %q is not within an approved folder", root)
	}

	return root, nil
}

const DefaultMaxFileSize = 1024 * 1024 // 1 MB

func (a *Agent) ScanApprovedFolders(opts scanner.Options) ([]scanner.Result, error) {
	folders := a.ListFolders()
	results := make([]scanner.Result, 0, len(folders))

	for _, folder := range folders {
		res, err := scanner.Scan(folder, opts)
		if err != nil {
			return nil, fmt.Errorf("scan folder %q: %w", folder, err)
		}
		results = append(results, res)
	}

	return results, nil
}

func (a *Agent) GetStatus(ctx context.Context, repoPath string) (git.Status, error) {
	root, err := a.ValidateRepository(ctx, repoPath)
	if err != nil {
		return git.Status{}, err
	}
	return a.GitClient.Status(ctx, root)
}

func (a *Agent) GetDiff(ctx context.Context, repoPath string) (string, error) {
	root, err := a.ValidateRepository(ctx, repoPath)
	if err != nil {
		return "", err
	}
	res, err := a.GitClient.Execute(ctx, root, "diff")
	if err != nil {
		return "", err
	}
	return res.Output, nil
}

func (a *Agent) ReadFile(ctx context.Context, repoPath string, relativePath string) ([]byte, error) {
	root, err := a.ValidateRepository(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	return scanner.ReadSelectedFile(root, relativePath, DefaultMaxFileSize)
}

func (a *Agent) Commit(ctx context.Context, repoPath string, opts git.CommitOptions) (git.OperationResult, error) {
	root, err := a.ValidateRepository(ctx, repoPath)
	if err != nil {
		return git.OperationResult{State: git.StateFailed, Error: err.Error()}, err
	}

	if err := a.Queue.AcquireLock(root, "commit"); err != nil {
		return git.OperationResult{State: git.StateFailed, Error: err.Error()}, err
	}
	defer a.Queue.ReleaseLock(root)

	if err := a.GitClient.StageFiles(ctx, root, opts.SelectedFiles); err != nil {
		return git.OperationResult{State: git.StateFailed, Error: err.Error()}, fmt.Errorf("stage files: %w", err)
	}

	return a.GitClient.Commit(ctx, root, opts)
}

func (a *Agent) Push(ctx context.Context, repoPath string, opts git.PushOptions) (git.OperationResult, error) {
	root, err := a.ValidateRepository(ctx, repoPath)
	if err != nil {
		return git.OperationResult{State: git.StateFailed, Error: err.Error()}, err
	}

	if err := a.Queue.AcquireLock(root, "push"); err != nil {
		return git.OperationResult{State: git.StateFailed, Error: err.Error()}, err
	}
	defer a.Queue.ReleaseLock(root)

	return a.GitClient.Push(ctx, root, opts)
}
