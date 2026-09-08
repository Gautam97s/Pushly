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
