package git

import (
	"fmt"
	"strings"
)

type Status struct {
	Branch   string
	Upstream string
	Ahead    int
	Behind   int
	Clean    bool
	Changes  []Change
}

type Change struct {
	Path         string
	OriginalPath string
	Index        ChangeState
	Worktree     ChangeState
	Kind         ChangeKind
}

type ChangeState string

const (
	StateUnchanged ChangeState = "unchanged"
	StateModified  ChangeState = "modified"
	StateAdded     ChangeState = "added"
	StateDeleted   ChangeState = "deleted"
	StateRenamed   ChangeState = "renamed"
	StateCopied    ChangeState = "copied"
	StateUntracked ChangeState = "untracked"
	StateIgnored   ChangeState = "ignored"
)

type ChangeKind string

const (
	KindModified  ChangeKind = "modified"
	KindAdded     ChangeKind = "added"
	KindDeleted   ChangeKind = "deleted"
	KindRenamed   ChangeKind = "renamed"
	KindCopied    ChangeKind = "copied"
	KindUntracked ChangeKind = "untracked"
	KindConflict  ChangeKind = "conflict"
)

func ParseStatus(output string) (Status, error) {
	var status Status
	records := strings.Split(output, "\x00")
	for index := 0; index < len(records); index++ {
		record := records[index]
		if record == "" {
			continue
		}
		if strings.HasPrefix(record, "## ") {
			parseShortBranchHeader(&status, strings.TrimPrefix(record, "## "))
			continue
		}
		if strings.HasPrefix(record, "# ") {
			if err := parseHeader(&status, strings.TrimPrefix(record, "# ")); err != nil {
				return Status{}, err
			}
			continue
		}
		if len(record) < 3 {
			return Status{}, fmt.Errorf("invalid status record %q", record)
		}

		change, err := parseChange(record)
		if err != nil {
			return Status{}, err
		}
		if change.Kind == KindRenamed || change.Kind == KindCopied {
			index++
			if index >= len(records) || records[index] == "" {
				return Status{}, fmt.Errorf("missing original path for %s", change.Kind)
			}
			change.OriginalPath = records[index]
		}
		status.Changes = append(status.Changes, change)
	}
	status.Clean = len(status.Changes) == 0
	return status, nil
}

func parseShortBranchHeader(status *Status, header string) {
	parts := strings.SplitN(header, "...", 2)
	status.Branch = parts[0]
	if len(parts) == 2 {
		upstreamAndTracking := parts[1]
		if bracket := strings.Index(upstreamAndTracking, " ["); bracket >= 0 {
			tracking := upstreamAndTracking[bracket:]
			upstreamAndTracking = upstreamAndTracking[:bracket]
			var value int
			if _, err := fmt.Sscanf(tracking, " [ahead %d", &value); err == nil {
				status.Ahead = value
			}
			if _, err := fmt.Sscanf(tracking, " [behind %d", &value); err == nil {
				status.Behind = value
			}
		}
		status.Upstream = upstreamAndTracking
		if strings.Contains(parts[1], ", behind ") {
			var ahead, behind int
			if _, err := fmt.Sscanf(parts[1], "%s [ahead %d, behind %d]", &status.Upstream, &ahead, &behind); err == nil {
				status.Ahead = ahead
				status.Behind = behind
			}
		}
	}
}

func parseHeader(status *Status, header string) error {
	switch {
	case strings.HasPrefix(header, "branch.head "):
		status.Branch = strings.TrimPrefix(header, "branch.head ")
	case strings.HasPrefix(header, "branch.upstream "):
		status.Upstream = strings.TrimPrefix(header, "branch.upstream ")
	case strings.HasPrefix(header, "branch.ab "):
		parts := strings.Fields(strings.TrimPrefix(header, "branch.ab "))
		for _, part := range parts {
			if len(part) < 2 {
				return fmt.Errorf("invalid branch tracking header %q", header)
			}
			var target *int
			switch part[0] {
			case '+':
				target = &status.Ahead
			case '-':
				target = &status.Behind
			default:
				return fmt.Errorf("invalid branch tracking value %q", part)
			}
			var value int
			if _, err := fmt.Sscanf(part[1:], "%d", &value); err != nil {
				return fmt.Errorf("invalid branch tracking value %q: %w", part, err)
			}
			*target = value
		}
	}
	return nil
}

func parseChange(record string) (Change, error) {
	indexCode, worktreeCode := record[0], record[1]
	path := record[3:]
	if path == "" {
		return Change{}, fmt.Errorf("missing path in status record %q", record)
	}

	change := Change{
		Path:     path,
		Index:    stateForCode(indexCode),
		Worktree: stateForCode(worktreeCode),
		Kind:     kindForCodes(indexCode, worktreeCode),
	}
	return change, nil
}

func stateForCode(code byte) ChangeState {
	switch code {
	case 'M':
		return StateModified
	case 'A':
		return StateAdded
	case 'D':
		return StateDeleted
	case 'R':
		return StateRenamed
	case 'C':
		return StateCopied
	case '?':
		return StateUntracked
	case '!':
		return StateIgnored
	default:
		return StateUnchanged
	}
}

func kindForCodes(indexCode, worktreeCode byte) ChangeKind {
	for _, code := range []byte{indexCode, worktreeCode} {
		switch code {
		case 'R':
			return KindRenamed
		case 'C':
			return KindCopied
		case 'D':
			return KindDeleted
		case 'A':
			return KindAdded
		case '?':
			return KindUntracked
		case 'U':
			return KindConflict
		}
	}
	return KindModified
}
