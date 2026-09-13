package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pushly/internal/agent"
	"pushly/internal/git"
	"pushly/internal/scanner"
)

const commandTimeout = 45 * time.Second

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	command := os.Args[1]

	ag, err := agent.NewAgent("", commandTimeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize agent: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	switch command {
	case "folder", "folders":
		handleFolderCommand(ag, os.Args[2:])

	case "scan":
		handleScanCommand(ctx, ag, os.Args[2:])

	case "status-json":
		requireArgCount(os.Args, 3, "usage: pushly status-json <repository-path>")
		repoPath := resolvePath(os.Args[2])
		status, err := ag.GitClient.Status(ctx, repoPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if err := json.NewEncoder(os.Stdout).Encode(status); err != nil {
			fmt.Fprintf(os.Stderr, "encode status error: %v\n", err)
			os.Exit(1)
		}

	case "status", "diff", "branch", "remote":
		requireArgCount(os.Args, 3, fmt.Sprintf("usage: pushly %s <repository-path>", command))
		repoPath := resolvePath(os.Args[2])
		result, err := ag.GitClient.Execute(ctx, repoPath, command)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(result.Output)

	case "commit":
		handleCommitCommand(ctx, ag, os.Args[2:])

	case "push":
		handlePushCommand(ctx, ag, os.Args[2:])

	default:
		printUsage()
		os.Exit(2)
	}
}

func handleFolderCommand(ag *agent.Agent, args []string) {
	if len(args) == 0 {
		fmt.Println("Approved folders:")
		for _, f := range ag.ListFolders() {
			fmt.Printf(" - %s\n", f)
		}
		return
	}

	subcommand := args[0]
	switch subcommand {
	case "list":
		for _, f := range ag.ListFolders() {
			fmt.Println(f)
		}

	case "add":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: pushly folder add <path>")
			os.Exit(2)
		}
		targetPath := resolvePath(args[1])
		if err := ag.AddFolder(targetPath); err != nil {
			fmt.Fprintf(os.Stderr, "error adding folder: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Added approved folder: %s\n", targetPath)

	case "remove", "rm":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: pushly folder remove <path>")
			os.Exit(2)
		}
		targetPath := resolvePath(args[1])
		if err := ag.RemoveFolder(targetPath); err != nil {
			fmt.Fprintf(os.Stderr, "error removing folder: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Removed approved folder: %s\n", targetPath)

	default:
		fmt.Fprintf(os.Stderr, "unknown folder subcommand %q; allowed: list, add, remove\n", subcommand)
		os.Exit(2)
	}
}

func handleScanCommand(ctx context.Context, ag *agent.Agent, args []string) {
	opts := scanner.Options{
		MaxDepth:        8,
		MaxRepositories: 1000,
	}

	var results []scanner.Result
	if len(args) > 0 {
		targetPath := resolvePath(args[0])
		res, err := scanner.Scan(targetPath, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scan error: %v\n", err)
			os.Exit(1)
		}
		results = []scanner.Result{res}
	} else {
		var err error
		results, err = ag.ScanApprovedFolders(opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scan error: %v\n", err)
			os.Exit(1)
		}
	}

	if err := json.NewEncoder(os.Stdout).Encode(results); err != nil {
		fmt.Fprintf(os.Stderr, "encode scan result: %v\n", err)
		os.Exit(1)
	}
}

func handleCommitCommand(ctx context.Context, ag *agent.Agent, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: pushly commit <repository-path> -m \"<message>\" [--snapshot <id>] <files...>")
		os.Exit(2)
	}

	repoPath := resolvePath(args[0])

	fs := flag.NewFlagSet("commit", flag.ContinueOnError)
	msg := fs.String("m", "", "Commit message")
	snapshotID := fs.String("snapshot", "", "Expected snapshot ID")

	if err := fs.Parse(args[1:]); err != nil {
		os.Exit(2)
	}

	files := fs.Args()
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "error: at least one file must be specified for commit")
		os.Exit(2)
	}

	opts := git.CommitOptions{
		SelectedFiles:      files,
		Message:            *msg,
		ExpectedSnapshotID: *snapshotID,
	}

	if err := opts.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "validation error: %v\n", err)
		os.Exit(2)
	}

	if err := ag.GitClient.StageFiles(ctx, repoPath, files); err != nil {
		fmt.Fprintf(os.Stderr, "stage error: %v\n", err)
		os.Exit(1)
	}

	res, err := ag.GitClient.Commit(ctx, repoPath, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "commit failed (%s): %v\n", res.State, err)
		os.Exit(1)
	}

	fmt.Printf("Commit successful:\n%s", res.Output)
}

func handlePushCommand(ctx context.Context, ag *agent.Agent, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: pushly push <repository-path> [<remote>] [<branch>]")
		os.Exit(2)
	}

	repoPath := resolvePath(args[0])
	var remote, branch string
	if len(args) > 1 {
		remote = args[1]
	}
	if len(args) > 2 {
		branch = args[2]
	}

	opts := git.PushOptions{
		Remote: remote,
		Branch: branch,
	}

	res, err := ag.GitClient.Push(ctx, repoPath, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "push failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Push successful:\n%s", res.Output)
}

func resolvePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid path %q: %v\n", path, err)
		os.Exit(1)
	}
	return filepath.Clean(abs)
}

func requireArgCount(args []string, expected int, usage string) {
	if len(args) != expected {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, strings.TrimSpace(`
Pushly - Remote Git Manager CLI

Usage:
  pushly folder <list|add|remove> [<path>]
  pushly scan [<path>]
  pushly status <repository-path>
  pushly status-json <repository-path>
  pushly diff <repository-path>
  pushly branch <repository-path>
  pushly remote <repository-path>
  pushly commit <repository-path> -m "<message>" [--snapshot <id>] <files...>
  pushly push <repository-path> [<remote>] [<branch>]
`))
}
