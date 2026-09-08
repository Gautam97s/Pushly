package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"pushly/internal/git"
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

	client := git.NewClient(commandTimeout)
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	result, err := client.Execute(ctx, repositoryPath, operation)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	fmt.Print(result.Output)
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: pushly <status|diff|branch|remote> <repository>")
}
