package agent

import (
	"fmt"
	"sync"
)

// OperationQueue manages serialization of operations per repository.
// It ensures that only one operation runs on a given repository at a time.
type OperationQueue struct {
	// mu protects access to repositories map
	mu           sync.RWMutex
	repositories map[string]*repoQueue
}

// repoQueue tracks the lock and active operation for a single repository
type repoQueue struct {
	mu        sync.Mutex
	locked    bool
	operation string
}

// NewOperationQueue creates a new operation queue manager
func NewOperationQueue() *OperationQueue {
	return &OperationQueue{
		repositories: make(map[string]*repoQueue),
	}
}

// AcquireLock acquires a lock for an operation on a repository.
// If the repository is already locked, it returns an error.
// The caller must call ReleaseLock when done.
func (oq *OperationQueue) AcquireLock(repositoryPath string, operationName string) error {
	oq.mu.Lock()
	rq, exists := oq.repositories[repositoryPath]
	if !exists {
		rq = &repoQueue{locked: false}
		oq.repositories[repositoryPath] = rq
	}
	oq.mu.Unlock()

	rq.mu.Lock()
	defer rq.mu.Unlock()

	if rq.locked {
		return fmt.Errorf("repository %q is busy with operation %q", repositoryPath, rq.operation)
	}

	rq.locked = true
	rq.operation = operationName
	return nil
}

// ReleaseLock releases the lock for a repository.
func (oq *OperationQueue) ReleaseLock(repositoryPath string) error {
	oq.mu.RLock()
	rq, exists := oq.repositories[repositoryPath]
	oq.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no lock registered for repository %q", repositoryPath)
	}

	rq.mu.Lock()
	defer rq.mu.Unlock()

	if !rq.locked {
		return fmt.Errorf("repository %q is not locked", repositoryPath)
	}

	rq.locked = false
	rq.operation = ""
	return nil
}

// IsLocked returns true if a repository is currently locked.
func (oq *OperationQueue) IsLocked(repositoryPath string) bool {
	oq.mu.RLock()
	rq, exists := oq.repositories[repositoryPath]
	oq.mu.RUnlock()

	if !exists {
		return false
	}

	rq.mu.Lock()
	defer rq.mu.Unlock()

	return rq.locked
}

// GetActiveOperation returns the name of the active operation for a repository,
// or an empty string if the repository is not locked.
func (oq *OperationQueue) GetActiveOperation(repositoryPath string) string {
	oq.mu.RLock()
	rq, exists := oq.repositories[repositoryPath]
	oq.mu.RUnlock()

	if !exists {
		return ""
	}

	rq.mu.Lock()
	defer rq.mu.Unlock()

	return rq.operation
}
