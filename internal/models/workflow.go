package models

import (
	"time"

	"github.com/google/go-github/v45/github"
)

// WorkflowRun represents a GitHub Actions workflow run
type WorkflowRun struct {
	ID           int64
	Name         string
	Status       string
	Conclusion   string
	StartedAt    time.Time
	CompletedAt  time.Time
}

// NewWorkflowRunFromGitHub creates a new WorkflowRun from a GitHub API response
func NewWorkflowRunFromGitHub(run *github.WorkflowRun) *WorkflowRun {
	return &WorkflowRun{
		ID:           run.GetID(),
		Name:         run.GetName(),
		Status:       run.GetStatus(),
		Conclusion:   run.GetConclusion(),
		StartedAt:    run.GetCreatedAt().Time,
		CompletedAt:  run.GetUpdatedAt().Time,
	}
}
