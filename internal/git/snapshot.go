package git

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

type Snapshot struct {
	ID     string
	Status Status
}

func NewSnapshot(status Status) Snapshot {
	canonical := cloneStatus(status)
	canonical.Changes = sortedChanges(canonical.Changes)

	data, _ := json.Marshal(canonical)
	hash := sha256.Sum256(data)
	return Snapshot{
		ID:     hex.EncodeToString(hash[:]),
		Status: canonical,
	}
}

func (snapshot Snapshot) IsStale(current Status) bool {
	return snapshot.ID != NewSnapshot(current).ID
}

func cloneStatus(status Status) Status {
	clone := status
	clone.Changes = append([]Change(nil), status.Changes...)
	return clone
}

func sortedChanges(changes []Change) []Change {
	sorted := append([]Change(nil), changes...)
	sort.Slice(sorted, func(left, right int) bool {
		if sorted[left].Path != sorted[right].Path {
			return sorted[left].Path < sorted[right].Path
		}
		return sorted[left].OriginalPath < sorted[right].OriginalPath
	})
	return sorted
}
