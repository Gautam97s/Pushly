package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pushly/internal/config"
	"pushly/internal/git"
)

type mockGitRunner struct {
	repoRoot string
	status   string
	output   string
	err      error
}

func (m *mockGitRunner) Run(_ context.Context, repoPath string, args []string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if len(args) > 0 && args[0] == "rev-parse" {
		return m.repoRoot, nil
	}
	if len(args) > 0 && args[0] == "status" {
		return m.status, nil
	}
	return m.output, nil
}

func TestAgentRejectsUnapprovedRepositories(t *testing.T) {
	tmpDir := t.TempDir()
	approvedFolder := filepath.Join(tmpDir, "Approved")
	unapprovedFolder := filepath.Join(tmpDir, "Unapproved")
	if err := os.MkdirAll(approvedFolder, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(unapprovedFolder, 0o755); err != nil {
		t.Fatal(err)
	}

	store := &config.FolderStore{}
	if err := store.AddFolder(approvedFolder); err != nil {
		t.Fatal(err)
	}

	runner := &mockGitRunner{repoRoot: unapprovedFolder}
	client := git.Client{Runner: runner, Timeout: time.Second}
	agent := NewAgentWithStore(store, client, nil)

	_, err := agent.GetStatus(context.Background(), unapprovedFolder)
	if err == nil {
		t.Fatal("expected GetStatus to fail on unapproved folder")
	}

	_, err = agent.Commit(context.Background(), unapprovedFolder, git.CommitOptions{
		SelectedFiles: []string{"file.txt"},
		Message:       "commit msg",
	})
	if err == nil {
		t.Fatal("expected Commit to fail on unapproved folder")
	}
}

func TestAgentExecutesCommitAndPushOnApprovedRepo(t *testing.T) {
	tmpDir := t.TempDir()
	approvedFolder := filepath.Join(tmpDir, "Approved")
	repoDir := filepath.Join(approvedFolder, "my-repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}

	store := &config.FolderStore{}
	if err := store.AddFolder(approvedFolder); err != nil {
		t.Fatal(err)
	}

	runner := &mockGitRunner{
		repoRoot: repoDir,
		status:   "# branch.head main\x00",
		output:   "[main 12345] test commit\n",
	}
	client := git.Client{Runner: runner, Timeout: time.Second}
	agent := NewAgentWithStore(store, client, nil)

	// Status check
	status, err := agent.GetStatus(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("GetStatus error: %v", err)
	}
	if status.Branch != "main" {
		t.Fatalf("branch = %q, want main", status.Branch)
	}

	// Commit
	commitRes, err := agent.Commit(context.Background(), repoDir, git.CommitOptions{
		SelectedFiles: []string{"file.txt"},
		Message:       "test commit",
	})
	if err != nil {
		t.Fatalf("Commit error: %v", err)
	}
	if !commitRes.Success {
		t.Fatalf("Commit failed: %+v", commitRes)
	}

	// Push
	pushRes, err := agent.Push(context.Background(), repoDir, git.PushOptions{
		Remote: "origin",
		Branch: "main",
	})
	if err != nil {
		t.Fatalf("Push error: %v", err)
	}
	if !pushRes.Success {
		t.Fatalf("Push failed: %+v", pushRes)
	}
}
