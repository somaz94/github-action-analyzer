package analyzer

import (
	"context"
	"fmt"
	"strings"
	"time"

	gh "github.com/google/go-github/v45/github"
	"github.com/somaz94/github-action-analyzer/internal/models"
)

// Analyzer handles workflow analysis
type Analyzer struct {
	client         GithubClient
	versionChecker VersionChecker
	debug          bool
}

// GithubClient interface defines methods for interacting with GitHub API
type GithubClient interface {
	GetWorkflowRuns(ctx context.Context, owner, repo, workflowFile string) ([]*gh.WorkflowRun, error)
	GetWorkflowJobLogs(ctx context.Context, owner, repo string, runID int64) (string, error)
	GetFileContent(ctx context.Context, owner, repo, path string) (string, error)
	GetLatestRelease(ctx context.Context, owner, repo string) (*gh.RepositoryRelease, error)
}

// VersionChecker interface for getting latest language versions
type VersionChecker interface {
	GetLatestVersion(lang string) (string, error)
}

// GitHubVersionChecker implements VersionChecker using GitHub API
type GitHubVersionChecker struct {
	client GithubClient
}

// GetLatestVersion retrieves the latest version for a given language
func (g *GitHubVersionChecker) GetLatestVersion(lang string) (string, error) {
	ctx := context.Background()
	switch lang {
	case "go":
		release, err := g.client.GetLatestRelease(ctx, "golang", "go")
		if err != nil {
			return "1.24", nil
		}
		version := strings.TrimPrefix(release.GetTagName(), "go")
		parts := strings.Split(version, ".")
		if len(parts) >= 2 {
			return parts[0] + "." + parts[1], nil
		}
		return "1.24", nil

	case "node":
		release, err := g.client.GetLatestRelease(ctx, "nodejs", "node")
		if err != nil {
			return "20.11", nil
		}
		version := strings.TrimPrefix(release.GetTagName(), "v")
		parts := strings.Split(version, ".")
		if len(parts) >= 2 {
			return parts[0] + "." + parts[1], nil
		}
		return "20.11", nil

	case "python":
		release, err := g.client.GetLatestRelease(ctx, "python", "cpython")
		if err != nil {
			return "3.12", nil
		}
		version := strings.TrimPrefix(release.GetTagName(), "v")
		if strings.Contains(version, "a") || strings.Contains(version, "b") || strings.Contains(version, "rc") {
			return "3.12", nil
		}
		parts := strings.Split(version, ".")
		if len(parts) >= 2 {
			return parts[0] + "." + parts[1], nil
		}
		return "3.12", nil

	default:
		return "", fmt.Errorf("unsupported language: %s", lang)
	}
}

// Language-specific cache strategies
var cacheStrategies = map[string][]models.CacheRecommendation{
	"go": {
		{
			Path:        "~/.cache/go-build",
			Description: "Cache Go build artifacts",
			Impact:      "Can reduce build time by up to 30%",
			Example: `      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '%s'
          cache: true  # This enables Go build cache`,
		},
		{
			Path:        "~/go/pkg/mod",
			Description: "Cache Go modules",
			Impact:      "Can reduce dependency download time significantly",
			Example: `      - name: Cache Go modules
        uses: actions/cache@v4
        with:
          path: |
            ~/go/pkg/mod
            ~/.cache/go-build
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-`,
		},
	},
	"node": {
		{
			Path:        "~/.npm",
			Description: "Cache npm dependencies",
			Impact:      "Can reduce npm install time by up to 50%",
			Example: `      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '%s'
          cache: 'npm'  # This enables npm cache`,
		},
	},
	"python": {
		{
			Path:        "~/.cache/pip",
			Description: "Cache pip dependencies",
			Impact:      "Can reduce pip install time significantly",
			Example: `      - name: Setup Python
        uses: actions/setup-python@v5
        with:
          python-version: '%s'
          cache: 'pip'  # This enables pip cache
          cache-dependency-path: |
            **/requirements.txt
            **/requirements-dev.txt`,
		},
	},
}

// NewAnalyzer creates a new instance of Analyzer
func NewAnalyzer(client GithubClient, debug bool) *Analyzer {
	return &Analyzer{
		client:         client,
		versionChecker: &GitHubVersionChecker{client: client},
		debug:          debug,
	}
}

// debugLog prints debug information if debug mode is enabled
func (a *Analyzer) debugLog(format string, args ...interface{}) {
	if a.debug {
		fmt.Printf(format+"\n", args...)
	}
}

// Analyze performs the workflow analysis
func (a *Analyzer) Analyze(ctx context.Context, owner, repo, workflowFile string) (*models.PerformanceReport, error) {
	report := &models.PerformanceReport{
		Repository:   fmt.Sprintf("%s/%s", owner, repo),
		WorkflowFile: workflowFile,
	}

	runs, err := a.client.GetWorkflowRuns(ctx, owner, repo, workflowFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow runs: %v", err)
	}

	if err := a.analyzeWorkflowRuns(ctx, owner, repo, runs, report); err != nil {
		return nil, err
	}

	if err := a.analyzeDockerConfigs(ctx, owner, repo, report); err != nil {
		return nil, err
	}

	if err := a.analyzeCaching(ctx, owner, repo, report); err != nil {
		return nil, err
	}

	a.generateCostSavingTips(report)

	return report, nil
}

// analyzeWorkflowRuns analyzes workflow execution history
func (a *Analyzer) analyzeWorkflowRuns(ctx context.Context, owner, repo string, runs []*gh.WorkflowRun, report *models.PerformanceReport) error {
	var totalTime time.Duration

	for _, githubRun := range runs {
		// Calculate actual workflow run time
		if githubRun.CreatedAt != nil && githubRun.UpdatedAt != nil {
			runDuration := githubRun.UpdatedAt.Sub(githubRun.CreatedAt.Time)
			totalTime += runDuration
		}

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

// analyzeDockerConfigs analyzes Dockerfile configurations
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

// analyzeCaching analyzes and suggests caching strategies
func (a *Analyzer) analyzeCaching(ctx context.Context, owner, repo string, report *models.PerformanceReport) error {
	workflowPath := report.WorkflowFile
	if !strings.HasPrefix(workflowPath, ".github/workflows/") {
		workflowPath = fmt.Sprintf(".github/workflows/%s", workflowPath)
	}

	workflowContent, err := a.client.GetFileContent(ctx, owner, repo, workflowPath)
	if err == nil {
		// Debug logging
		a.debugLog("Workflow content:\n%s", workflowContent)

		detectedLangs := detectLanguagesFromWorkflow(workflowContent)
		a.debugLog("Detected languages: %v", detectedLangs)

		for _, lang := range detectedLangs {
			latestVersion, err := a.versionChecker.GetLatestVersion(lang)
			if err != nil {
				a.debugLog("Error getting latest version for %s: %v", lang, err)
				continue
			}
			a.debugLog("Latest version for %s: %s", lang, latestVersion)

			if strategies, ok := cacheStrategies[lang]; ok {
				for _, strategy := range strategies {
					updatedStrategy := strategy
					updatedStrategy.Example = fmt.Sprintf(strategy.Example, latestVersion)
					report.CacheRecommendations = append(report.CacheRecommendations, updatedStrategy)
				}
			}
		}
	} else {
		a.debugLog("Error getting workflow content: %v", err)
	}

	return nil
}

// generateCostSavingTips generates cost optimization recommendations
func (a *Analyzer) generateCostSavingTips(report *models.PerformanceReport) {
	tips := []string{
		"Consider using GitHub Actions cache to speed up dependencies installation",
		"Use matrix builds for parallel execution",
		"Implement proper Docker layer caching",
		fmt.Sprintf("Total execution time: %v - Consider optimizing long-running steps", report.TotalExecutionTime),
	}
	report.CostSavingTips = tips
}

// analyzeSteps analyzes individual workflow steps
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

// analyzeDockerfile analyzes Dockerfile for optimizations
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

// detectLanguagesFromWorkflow detects programming languages used in workflow
func detectLanguagesFromWorkflow(content string) []string {
	var languages []string
	languagePatterns := map[string][]string{
		"go": {
			"go build",
			"go test",
			"setup-go",
			"actions/setup-go",
			"go-version",
			"uses: actions/setup-go",
		},
		"node": {
			"npm",
			"yarn",
			"setup-node",
			"actions/setup-node",
			"package.json",
			"node-version",
			"uses: actions/setup-node",
		},
		"python": {
			"pip",
			"python",
			"setup-python",
			"actions/setup-python",
			"requirements.txt",
			"python-version",
			"uses: actions/setup-python",
		},
	}

	lowerContent := strings.ToLower(content)

	for lang, patterns := range languagePatterns {
		for _, pattern := range patterns {
			lowerPattern := strings.ToLower(pattern)
			if strings.Contains(lowerContent, lowerPattern) {
				languages = append(languages, lang)
				break
			}
		}
	}

	return unique(languages)
}

// unique removes duplicate entries from a string slice
func unique(slice []string) []string {
	keys := make(map[string]bool)
	var list []string
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// Escape GitHub expression in example strings
func escapeGitHubExpression(example string) string {
	return strings.ReplaceAll(example, "${{", "\\${\\{")
}
