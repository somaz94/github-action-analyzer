package models

import (
	"time"

	"github.com/google/go-github/v45/github"
)

// WorkflowRun represents a GitHub Actions workflow run
type WorkflowRun struct {
	ID          int64
	Name        string
	Status      string
	Conclusion  string
	StartedAt   time.Time
	CompletedAt time.Time
}

// NewWorkflowRunFromGitHub creates a new WorkflowRun from a GitHub API response
func NewWorkflowRunFromGitHub(run *github.WorkflowRun) *WorkflowRun {
	return &WorkflowRun{
		ID:          run.GetID(),
		Name:        run.GetName(),
		Status:      run.GetStatus(),
		Conclusion:  run.GetConclusion(),
		StartedAt:   run.GetCreatedAt().Time,
		CompletedAt: run.GetUpdatedAt().Time,
	}
}

// WorkflowAnalysis represents workflow-specific analysis
type WorkflowAnalysis struct {
	ParallelJobs        bool     `json:"parallel_jobs"`
	MatrixStrategy      bool     `json:"matrix_strategy"`
	Recommendations     []string `json:"recommendations"`
	RunnerOptimizations []string `json:"runner_optimizations"`
	SecurityTips        []string `json:"security_tips"`
}

// WorkflowJob represents a job in the workflow
type WorkflowJob struct {
	Name         string   `json:"name"`
	RunsOn       string   `json:"runs_on"`
	Dependencies []string `json:"dependencies"`
	UsesMatrix   bool     `json:"uses_matrix"`
}
