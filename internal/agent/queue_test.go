package agent

import (
	"testing"
)

func TestOperationQueueSerializesOperations(t *testing.T) {
	queue := NewOperationQueue()
	repoPath := "C:/Projects/TestRepo"

	// First operation should succeed
	if err := queue.AcquireLock(repoPath, "commit"); err != nil {
		t.Fatalf("AcquireLock for commit failed: %v", err)
	}

	// Verify it's locked
	if !queue.IsLocked(repoPath) {
		t.Fatalf("Repository should be locked")
	}

	// Second operation on same repo should fail
	if err := queue.AcquireLock(repoPath, "push"); err == nil {
		t.Fatalf("AcquireLock for push should have failed when repo is locked")
	}

	// Release the lock
	if err := queue.ReleaseLock(repoPath); err != nil {
		t.Fatalf("ReleaseLock failed: %v", err)
	}

	// Verify it's unlocked
	if queue.IsLocked(repoPath) {
		t.Fatalf("Repository should be unlocked")
	}

	// Now the second operation should succeed
	if err := queue.AcquireLock(repoPath, "push"); err != nil {
		t.Fatalf("AcquireLock for push failed: %v", err)
	}

	if err := queue.ReleaseLock(repoPath); err != nil {
		t.Fatalf("ReleaseLock failed: %v", err)
	}
}

func TestOperationQueueAllowsMultipleRepositories(t *testing.T) {
	queue := NewOperationQueue()
	repo1 := "C:/Projects/Repo1"
	repo2 := "C:/Projects/Repo2"

	// Lock both repositories
	if err := queue.AcquireLock(repo1, "commit"); err != nil {
		t.Fatalf("AcquireLock for repo1 failed: %v", err)
	}

	if err := queue.AcquireLock(repo2, "push"); err != nil {
		t.Fatalf("AcquireLock for repo2 failed: %v", err)
	}

	// Both should be locked
	if !queue.IsLocked(repo1) || !queue.IsLocked(repo2) {
		t.Fatalf("Both repositories should be locked")
	}

	// Release repo1
	if err := queue.ReleaseLock(repo1); err != nil {
		t.Fatalf("ReleaseLock for repo1 failed: %v", err)
	}

	// repo1 should be unlocked, repo2 should still be locked
	if queue.IsLocked(repo1) {
		t.Fatalf("repo1 should be unlocked")
	}
	if !queue.IsLocked(repo2) {
		t.Fatalf("repo2 should still be locked")
	}

	if err := queue.ReleaseLock(repo2); err != nil {
		t.Fatalf("ReleaseLock for repo2 failed: %v", err)
	}
}

func TestOperationQueueTracksOperationNames(t *testing.T) {
	queue := NewOperationQueue()
	repoPath := "C:/Projects/TestRepo"

	// Initially no operation
	if op := queue.GetActiveOperation(repoPath); op != "" {
		t.Fatalf("GetActiveOperation should return empty string, got %q", op)
	}

	// Acquire lock with commit operation
	if err := queue.AcquireLock(repoPath, "commit"); err != nil {
		t.Fatalf("AcquireLock failed: %v", err)
	}

	// Should return commit
	if op := queue.GetActiveOperation(repoPath); op != "commit" {
		t.Fatalf("GetActiveOperation should return 'commit', got %q", op)
	}

	// Release lock
	queue.ReleaseLock(repoPath)

	// Should return empty string again
	if op := queue.GetActiveOperation(repoPath); op != "" {
		t.Fatalf("GetActiveOperation should return empty string after release, got %q", op)
	}
}

func TestOperationQueueErrorHandling(t *testing.T) {
	queue := NewOperationQueue()
	repoPath := "C:/Projects/TestRepo"

	// ReleaseLock on non-existent lock should fail
	if err := queue.ReleaseLock(repoPath); err == nil {
		t.Fatalf("ReleaseLock on non-existent lock should fail")
	}

	// Acquire lock
	queue.AcquireLock(repoPath, "commit")

	// ReleaseLock should succeed
	if err := queue.ReleaseLock(repoPath); err != nil {
		t.Fatalf("ReleaseLock should succeed: %v", err)
	}

	// ReleaseLock again should fail
	if err := queue.ReleaseLock(repoPath); err == nil {
		t.Fatalf("Double ReleaseLock should fail")
	}
}
