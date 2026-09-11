package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"pushly/internal/git"
	"pushly/internal/scanner"
)

const commandTimeout = 30 * time.Second

func main() {
	if len(os.Args) != 3 {
		printUsage()
		os.Exit(2)
	}

	operation := os.Args[1]
	repositoryPath, err := filepath.Abs(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid repository path: %v\n", err)
		os.Exit(1)
	}

	if operation == "scan" {
		result, err := scanner.Scan(repositoryPath, scanner.Options{
			MaxDepth:        8,
			MaxRepositories: 1000,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "encode scan result: %v\n", err)
			os.Exit(1)
		}
		return
	}

	client := git.NewClient(commandTimeout)
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	if operation == "status-json" {
		status, err := client.Status(ctx, repositoryPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		if err := json.NewEncoder(os.Stdout).Encode(status); err != nil {
			fmt.Fprintf(os.Stderr, "encode status: %v\n", err)
			os.Exit(1)
		}
		return
	}

	result, err := client.Execute(ctx, repositoryPath, operation)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	fmt.Print(result.Output)
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: pushly <status|status-json|diff|branch|remote|scan> <path>")
}
