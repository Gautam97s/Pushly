package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Result struct {
	Output string
}

type Runner interface {
	Run(context.Context, string, []string) (string, error)
}

type osRunner struct{}

func (osRunner) Run(ctx context.Context, repositoryPath string, arguments []string) (string, error) {
	command := exec.CommandContext(ctx, "git", arguments...)
	command.Dir = repositoryPath
	output, err := command.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("git command timed out: %w", ctx.Err())
		}
		message := strings.TrimSpace(string(output))
		if message == "" {
			return "", fmt.Errorf("git command failed: %w", err)
		}
		return "", fmt.Errorf("git command failed: %s", message)
	}

	return string(output), nil
}

type Client struct {
	Runner  Runner
	Timeout time.Duration
}

func NewClient(timeout time.Duration) Client {
	return Client{Runner: osRunner{}, Timeout: timeout}
}

func (c Client) Execute(ctx context.Context, repositoryPath, operation string) (Result, error) {
	arguments, ok := allowedArguments(operation)
	if !ok {
		return Result{}, fmt.Errorf("unsupported operation %q; allowed operations: status, diff, branch, remote", operation)
	}

	if c.Runner == nil {
		return Result{}, fmt.Errorf("git runner is not configured")
	}

	commandContext, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	output, err := c.Runner.Run(commandContext, repositoryPath, arguments)
	if err != nil {
		return Result{}, err
	}

	return Result{Output: output}, nil
}

func allowedArguments(operation string) ([]string, bool) {
	commands := map[string][]string{
		"status": {"status", "--short", "--branch"},
		"diff":   {"diff", "--no-ext-diff", "--no-color"},
		"branch": {"branch", "--show-current"},
		"remote": {"remote", "-v"},
	}

	arguments, ok := commands[operation]
	return arguments, ok
}
