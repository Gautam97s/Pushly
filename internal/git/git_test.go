package git

import (
	"context"
	"strings"
	"testing"
	"time"
)

type fakeRunner struct {
	arguments []string
	output    string
	err       error
}

func (f *fakeRunner) Run(_ context.Context, _ string, arguments []string) (string, error) {
	f.arguments = arguments
	return f.output, f.err
}

func TestClientAllowsOnlyReadOperations(t *testing.T) {
	runner := &fakeRunner{output: "## main\n"}
	client := Client{Runner: runner, Timeout: time.Second}

	result, err := client.Execute(context.Background(), t.TempDir(), "status")
	if err != nil {
		t.Fatalf("Execute returned an error: %v", err)
	}
	if result.Output != runner.output {
		t.Fatalf("output = %q, want %q", result.Output, runner.output)
	}
	if strings.Join(runner.arguments, " ") != "status --short --branch" {
		t.Fatalf("arguments = %v", runner.arguments)
	}
}

func TestClientRejectsUnsupportedOperations(t *testing.T) {
	runner := &fakeRunner{}
	client := Client{Runner: runner, Timeout: time.Second}

	_, err := client.Execute(context.Background(), t.TempDir(), "shell")
	if err == nil {
		t.Fatal("Execute succeeded for an unsupported operation")
	}
	if !strings.Contains(err.Error(), "unsupported operation") {
		t.Fatalf("error = %q", err)
	}
	if runner.arguments != nil {
		t.Fatalf("runner was called with %v", runner.arguments)
	}
}

func TestParseStatus(t *testing.T) {
	output := "# branch.head main\x00# branch.upstream origin/main\x00# branch.ab +2 -1\x00"
	output += " M README.md\x00"
	output += "?? file with spaces.txt\x00"
	output += "R. renamed.txt\x00old.txt\x00"

	status, err := ParseStatus(output)
	if err != nil {
		t.Fatalf("ParseStatus returned an error: %v", err)
	}
	if status.Branch != "main" || status.Upstream != "origin/main" || status.Ahead != 2 || status.Behind != 1 {
		t.Fatalf("unexpected branch metadata: %+v", status)
	}
	if len(status.Changes) != 3 {
		t.Fatalf("changes = %d, want 3", len(status.Changes))
	}
	if status.Changes[1].Kind != KindUntracked || status.Changes[1].Path != "file with spaces.txt" {
		t.Fatalf("unexpected untracked change: %+v", status.Changes[1])
	}
	if status.Changes[2].Kind != KindRenamed || status.Changes[2].OriginalPath != "old.txt" {
		t.Fatalf("unexpected rename: %+v", status.Changes[2])
	}
}

func TestParseShortBranchHeader(t *testing.T) {
	status, err := ParseStatus("## dev...origin/main [ahead 2, behind 1]\x00")
	if err != nil {
		t.Fatalf("ParseStatus returned an error: %v", err)
	}
	if status.Branch != "dev" || status.Upstream != "origin/main" || status.Ahead != 2 || status.Behind != 1 {
		t.Fatalf("unexpected branch metadata: %+v", status)
	}
}

func TestClientStatusUsesMachineReadableOutput(t *testing.T) {
	runner := &fakeRunner{output: "# branch.head main\x00"}
	client := Client{Runner: runner, Timeout: time.Second}

	status, err := client.Status(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("Status returned an error: %v", err)
	}
	if status.Branch != "main" {
		t.Fatalf("branch = %q, want main", status.Branch)
	}
	if strings.Join(runner.arguments, " ") != "status --porcelain=v1 -z -b" {
		t.Fatalf("arguments = %v", runner.arguments)
	}
}

func TestClientRepositoryRoot(t *testing.T) {
	runner := &fakeRunner{output: "C:/Projects/App\n"}
	client := Client{Runner: runner, Timeout: time.Second}

	root, err := client.RepositoryRoot(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("RepositoryRoot returned an error: %v", err)
	}
	if root != "C:/Projects/App" {
		t.Fatalf("root = %q, want C:/Projects/App", root)
	}
	if strings.Join(runner.arguments, " ") != "rev-parse --show-toplevel" {
		t.Fatalf("arguments = %v", runner.arguments)
	}
}

func TestSnapshotIsStableAndDetectsChanges(t *testing.T) {
	status := Status{
		Branch: "main",
		Changes: []Change{
			{Path: "z.txt", Kind: KindModified},
			{Path: "a.txt", Kind: KindUntracked},
		},
	}
	first := NewSnapshot(status)
	status.Changes[0].Path = "changed.txt"
	second := NewSnapshot(status)

	if first.ID == "" || first.ID == second.ID {
		t.Fatalf("snapshot IDs = %q and %q, want distinct IDs", first.ID, second.ID)
	}
	if !first.IsStale(status) {
		t.Fatal("snapshot was not marked stale after a change")
	}
	if first.Status.Changes[1].Path != "z.txt" {
		t.Fatalf("snapshot changed after source mutation: %+v", first.Status.Changes)
	}
}

func TestSnapshotIgnoresChangeOrder(t *testing.T) {
	first := NewSnapshot(Status{Changes: []Change{
		{Path: "b.txt", Kind: KindModified},
		{Path: "a.txt", Kind: KindAdded},
	}})
	second := NewSnapshot(Status{Changes: []Change{
		{Path: "a.txt", Kind: KindAdded},
		{Path: "b.txt", Kind: KindModified},
	}})

	if first.ID != second.ID {
		t.Fatalf("snapshot IDs differ for reordered changes: %q != %q", first.ID, second.ID)
	}
}
