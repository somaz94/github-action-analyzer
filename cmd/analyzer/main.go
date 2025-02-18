package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/somaz94/github-action-analyzer/internal/analyzer"
	"github.com/somaz94/github-action-analyzer/internal/github"
)

func main() {
	ctx := context.Background()

	// Get inputs from environment variables
	token := os.Getenv("INPUT_GITHUB_TOKEN")
	workflowFile := os.Getenv("INPUT_WORKFLOW_FILE")
	repository := os.Getenv("INPUT_REPOSITORY")

	if token == "" || workflowFile == "" || repository == "" {
		log.Fatal("Required inputs are missing")
	}

	// Parse repository owner and name
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		log.Fatal("Invalid repository format. Expected: owner/repo")
	}
	owner, repo := parts[0], parts[1]

	// Initialize GitHub client
	client := github.NewClient(token)

	// Create analyzer
	analyzer := analyzer.NewAnalyzer(client)

	// Run analysis
	report, err := analyzer.Analyze(ctx, owner, repo, workflowFile)
	if err != nil {
		log.Fatalf("Analysis failed: %v", err)
	}

	// Output report
	if err := report.Output(); err != nil {
		log.Fatalf("Failed to output report: %v", err)
	}
}
