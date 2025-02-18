package analyzer

import (
	"context"
	"fmt"
	"strings"
	"time"

	gh "github.com/google/go-github/v45/github"
	"github.com/somaz94/github-action-analyzer/internal/models"
)

type Analyzer struct {
	client GithubClient
}

type GithubClient interface {
	GetWorkflowRuns(ctx context.Context, owner, repo, workflowFile string) ([]*gh.WorkflowRun, error)
	GetWorkflowJobLogs(ctx context.Context, owner, repo string, runID int64) (string, error)
	GetFileContent(ctx context.Context, owner, repo, path string) (string, error)
}

func NewAnalyzer(client GithubClient) *Analyzer {
	return &Analyzer{
		client: client,
	}
}

func (a *Analyzer) Analyze(ctx context.Context, owner, repo, workflowFile string) (*models.PerformanceReport, error) {
	report := &models.PerformanceReport{
		Repository:   fmt.Sprintf("%s/%s", owner, repo),
		WorkflowFile: workflowFile,
	}

	// Get workflow runs
	runs, err := a.client.GetWorkflowRuns(ctx, owner, repo, workflowFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow runs: %v", err)
	}

	// Analyze workflow runs
	if err := a.analyzeWorkflowRuns(ctx, owner, repo, runs, report); err != nil {
		return nil, err
	}

	// Analyze Docker configurations
	if err := a.analyzeDockerConfigs(ctx, owner, repo, report); err != nil {
		return nil, err
	}

	// Analyze caching strategies
	if err := a.analyzeCaching(ctx, runs, report); err != nil {
		return nil, err
	}

	// Generate cost saving tips
	a.generateCostSavingTips(report)

	return report, nil
}

func (a *Analyzer) analyzeWorkflowRuns(ctx context.Context, owner, repo string, runs []*gh.WorkflowRun, report *models.PerformanceReport) error {
	var totalTime time.Duration

	for _, githubRun := range runs {
		run := models.NewWorkflowRunFromGitHub(githubRun)

		// Get job logs
		logs, err := a.client.GetWorkflowJobLogs(ctx, owner, repo, run.ID)
		if err != nil {
			return fmt.Errorf("failed to get job logs: %v", err)
		}

		// Analyze steps
		steps, duration := analyzeSteps(logs)
		totalTime += duration

		// Identify slow steps
		for _, step := range steps {
			if step.ExecutionTime > 5*time.Minute {
				report.SlowSteps = append(report.SlowSteps, step)
			}
		}
	}

	report.TotalExecutionTime = totalTime
	return nil
}

func (a *Analyzer) analyzeDockerConfigs(ctx context.Context, owner, repo string, report *models.PerformanceReport) error {
	// Analyze Dockerfile if exists
	dockerFile, err := a.client.GetFileContent(ctx, owner, repo, "Dockerfile")
	if err != nil {
		return nil // Dockerfile might not exist
	}

	optimizations := analyzeDockerfile(dockerFile)
	report.DockerOptimizations = optimizations
	return nil
}

func (a *Analyzer) analyzeCaching(_ context.Context, runs []*gh.WorkflowRun, report *models.PerformanceReport) error {
	for _, run := range runs {
		// Analyze cache usage and get recommendations
		recommendations := analyzeCacheHitPatterns(run)
		if len(recommendations) > 0 {
			report.CacheRecommendations = append(report.CacheRecommendations, recommendations...)
		}
	}
	return nil
}

func (a *Analyzer) generateCostSavingTips(report *models.PerformanceReport) {
	tips := []string{
		"Consider using GitHub Actions cache to speed up dependencies installation",
		"Use matrix builds for parallel execution",
		"Implement proper Docker layer caching",
		fmt.Sprintf("Total execution time: %v - Consider optimizing long-running steps", report.TotalExecutionTime),
	}
	report.CostSavingTips = tips
}

// analyzeSteps parses workflow logs and returns step analysis
func analyzeSteps(logs string) ([]models.StepAnalysis, time.Duration) {
	var steps []models.StepAnalysis
	var totalDuration time.Duration

	// Parse logs to extract step information
	// This is a simple implementation - you might want to enhance this
	lines := strings.Split(logs, "\n")
	var currentStep string
	var stepStartTime time.Time

	for _, line := range lines {
		if strings.Contains(line, "##[group]") {
			// New step started
			if currentStep != "" {
				duration := time.Since(stepStartTime)
				steps = append(steps, models.StepAnalysis{
					Name:          currentStep,
					ExecutionTime: duration,
					IsSlowStep:    duration > 5*time.Minute,
				})
				totalDuration += duration
			}
			currentStep = strings.TrimPrefix(line, "##[group]")
			stepStartTime = time.Now()
		}
	}

	return steps, totalDuration
}

// analyzeDockerfile analyzes Dockerfile content for optimization opportunities
func analyzeDockerfile(content string) []models.DockerOptimization {
	var optimizations []models.DockerOptimization

	// Check for multi-stage builds
	if !strings.Contains(content, "FROM") || strings.Count(content, "FROM") < 2 {
		optimizations = append(optimizations, models.DockerOptimization{
			Issue:       "No multi-stage build detected",
			Suggestion:  "Consider using multi-stage builds to reduce final image size",
			Improvement: "Can reduce image size by up to 50%",
		})
	}

	// Check for layer caching
	if !strings.Contains(content, "COPY --from") {
		optimizations = append(optimizations, models.DockerOptimization{
			Issue:       "No layer caching strategy detected",
			Suggestion:  "Implement proper layer caching by copying only necessary files",
			Improvement: "Can improve build time significantly",
		})
	}

	return optimizations
}

// analyzeCacheHitPatterns analyzes workflow run for cache usage patterns
func analyzeCacheHitPatterns(run *gh.WorkflowRun) []models.CacheRecommendation {
	var recommendations []models.CacheRecommendation

	// Check workflow event type
	event := run.GetEvent()
	if event == "push" || event == "pull_request" {
		// Add Go-specific cache recommendations
		recommendations = append(recommendations, models.CacheRecommendation{
			Path:        "~/.cache/go-build",
			Description: "Cache Go build artifacts",
			Impact:      "Can reduce build time by up to 30%",
		})

		recommendations = append(recommendations, models.CacheRecommendation{
			Path:        "~/go/pkg/mod",
			Description: "Cache Go modules",
			Impact:      "Can reduce dependency download time significantly",
		})
	}

	return recommendations
}
