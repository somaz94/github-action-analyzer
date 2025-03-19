package analyzer

import (
	"context"
	"fmt"
	"os"
	"strconv"
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
		if strings.Contains(version, "nightly") || strings.Contains(version, "test") {
			return "20.11", nil
		}
		parts := strings.Split(version, ".")
		if len(parts) >= 2 {
			majorVer, _ := strconv.Atoi(parts[0])
			if majorVer > 20 {
				return "20.11", nil
			}
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

	case "java":
		release, err := g.client.GetLatestRelease(ctx, "adoptium", "temurin")
		if err != nil {
			return "17", nil
		}
		version := strings.TrimPrefix(release.GetTagName(), "jdk-")
		parts := strings.Split(version, ".")
		if len(parts) >= 1 {
			return parts[0], nil
		}
		return "17", nil

	case "ruby":
		release, err := g.client.GetLatestRelease(ctx, "ruby", "ruby")
		if err != nil {
			return "3.2", nil
		}
		version := strings.TrimPrefix(release.GetTagName(), "v")
		parts := strings.Split(version, ".")
		if len(parts) >= 2 {
			return parts[0] + "." + parts[1], nil
		}
		return "3.2", nil

	case "rust":
		// Rust uses a different versioning system, returning latest stable
		return "stable", nil

	case "dotnet":
		release, err := g.client.GetLatestRelease(ctx, "dotnet", "core")
		if err != nil {
			return "7.0", nil
		}
		version := strings.TrimPrefix(release.GetTagName(), "v")
		parts := strings.Split(version, ".")
		if len(parts) >= 2 {
			return parts[0] + "." + parts[1], nil
		}
		return "7.0", nil

	default:
		return "", fmt.Errorf("unsupported language: %s", lang)
	}
}

// Language-specific cache strategies
var cacheStrategies = map[string][]models.CacheRecommendation{
	"go": {
		{
			Path:        "~/.cache/go-build",
			Description: "Cache Go build artifacts and modules",
			Impact:      "Can reduce build time and dependency download time significantly",
			Example: `      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '%s'
          cache: true  # This enables Go build cache

      - uses: actions/cache@v4
        with:
          path: |
            ~/.cache/go-build
            ~/go/pkg/mod
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
			Example: `      - name: Get npm cache directory
        id: npm-cache-dir
        shell: bash
        run: echo "dir=$(npm config get cache)" >> ${GITHUB_OUTPUT}

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '%s'
          cache: 'npm'  # This enables npm cache

      - uses: actions/cache@v4
        id: npm-cache
        with:
          path: ${{ steps.npm-cache-dir.outputs.dir }}
          key: ${{ runner.os }}-node-${{ hashFiles('**/package-lock.json') }}
          restore-keys: |
            ${{ runner.os }}-node-`,
		},
		{
			Path:        "node_modules",
			Description: "Cache node_modules directory",
			Impact:      "Can significantly reduce installation time for large projects",
			Example: `      - uses: actions/cache@v4
        with:
          path: '**/node_modules'
          key: ${{ runner.os }}-modules-${{ hashFiles('**/package-lock.json') }}`,
		},
	},
	"python": {
		{
			Path:        "~/.cache/pip",
			Description: "Cache pip dependencies",
			Impact:      "Can reduce pip install time significantly",
			Example: `      - name: Set up Python
        id: setup-python
        uses: actions/setup-python@v5
        with:
          python-version: '%s'
          cache: 'pip'
          cache-dependency-path: |
            **/requirements.txt
            **/requirements-dev.txt

      - uses: actions/cache@v4
        with:
          path: |
            ~/.cache/pip
            ~/.local/share/virtualenvs
          key: ${{ runner.os }}-python-${{ hashFiles('**/requirements.txt') }}
          restore-keys: |
            ${{ runner.os }}-python-`,
		},
	},
	"java": {
		{
			Path:        "~/.m2/repository",
			Description: "Cache Maven dependencies",
			Impact:      "Can significantly reduce build time by caching Maven dependencies",
			Example: `      - name: Set up Java
        uses: actions/setup-java@v4
        with:
          java-version: '%s'
          distribution: 'temurin'
          cache: 'maven'

      - uses: actions/cache@v4
        with:
          path: ~/.m2/repository
          key: ${{ runner.os }}-maven-${{ hashFiles('**/pom.xml') }}
          restore-keys: |
            ${{ runner.os }}-maven-`,
		},
		{
			Path:        "~/.gradle",
			Description: "Cache Gradle dependencies and wrapper",
			Impact:      "Can significantly reduce build time by caching Gradle dependencies",
			Example: `      - name: Set up Java
        uses: actions/setup-java@v4
        with:
          java-version: '%s'
          distribution: 'temurin'
          cache: 'gradle'

      - uses: actions/cache@v4
        with:
          path: |
            ~/.gradle/caches
            ~/.gradle/wrapper
          key: ${{ runner.os }}-gradle-${{ hashFiles('**/*.gradle*', '**/gradle-wrapper.properties') }}
          restore-keys: |
            ${{ runner.os }}-gradle-`,
		},
	},
	"ruby": {
		{
			Path:        "vendor/bundle",
			Description: "Cache Ruby gems using Bundler",
			Impact:      "Can reduce gem installation time significantly",
			Example: `      - name: Set up Ruby
        uses: ruby/setup-ruby@v1
        with:
          ruby-version: '%s'
          bundler-cache: true  # This handles caching automatically`,
		},
	},
	"rust": {
		{
			Path:        "~/.cargo",
			Description: "Cache Rust dependencies and build artifacts",
			Impact:      "Can significantly reduce build time by caching Cargo dependencies and compiled artifacts",
			Example: `      - name: Set up Rust
        uses: dtolnay/rust-toolchain@stable
        with:
          toolchain: '%s'

      - uses: actions/cache@v4
        with:
          path: |
            ~/.cargo/bin/
            ~/.cargo/registry/index/
            ~/.cargo/registry/cache/
            ~/.cargo/git/db/
            target/
          key: ${{ runner.os }}-cargo-${{ hashFiles('**/Cargo.lock') }}`,
		},
	},
	"dotnet": {
		{
			Path:        ".dotnet",
			Description: "Cache .NET SDK installation",
			Impact:      "Can significantly reduce setup time by caching the .NET SDK",
			Example: `      - name: Cache .NET SDK
        uses: actions/cache@v4
        with:
          path: .\.dotnet
          key: ${{ runner.os }}-dotnet-${{ hashFiles('**/*.csproj') }}
          restore-keys: |
            ${{ runner.os }}-dotnet-

      - name: Setup .NET
        uses: actions/setup-dotnet@v4
        with:
          dotnet-version: '%s'
          cache: true

      - name: Cache NuGet packages
        uses: actions/cache@v4
        with:
          path: ~/.nuget/packages
          key: ${{ runner.os }}-nuget-${{ hashFiles('**/*.csproj') }}
          restore-keys: |
            ${{ runner.os }}-nuget-`,
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
	// Parse timeout from env
	timeoutStr := os.Getenv("TIMEOUT")
	timeout := 60 * time.Minute // default timeout changed to 60 minutes
	if t, err := strconv.Atoi(timeoutStr); err == nil && t > 0 {
		timeout = time.Duration(t) * time.Minute
	}

	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	report := &models.PerformanceReport{
		Repository:   fmt.Sprintf("%s/%s", owner, repo),
		WorkflowFile: workflowFile,
	}

	// Run analysis tasks with timeout context
	errCh := make(chan error, 1)
	go func() {
		var err error
		defer func() {
			errCh <- err
		}()

		if err = a.analyzeWorkflowRuns(ctx, owner, repo, workflowFile, report); err != nil {
			return
		}
		if err = a.analyzeDockerConfigs(ctx, owner, repo, report); err != nil {
			return
		}
		if err = a.analyzeCaching(ctx, owner, repo, report); err != nil {
			return
		}

		// Get workflow content for structure analysis
		workflowPath := report.WorkflowFile
		if !strings.HasPrefix(workflowPath, ".github/workflows/") {
			workflowPath = fmt.Sprintf(".github/workflows/%s", workflowPath)
		}

		if content, err := a.client.GetFileContent(ctx, owner, repo, workflowPath); err == nil {
			if err = a.analyzeWorkflowStructure(content, report); err != nil {
				a.debugLog("Warning: workflow structure analysis failed: %v", err)
			}
		}

		a.generateCostSavingTips(report)
	}()

	// Wait for either completion or timeout
	select {
	case err := <-errCh:
		if err != nil {
			return nil, fmt.Errorf("analysis failed: %v", err)
		}
		return report, nil
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("analysis timed out after %v minutes", timeout.Minutes())
		}
		return nil, ctx.Err()
	}
}

// analyzeWorkflowRuns analyzes workflow execution history
func (a *Analyzer) analyzeWorkflowRuns(ctx context.Context, owner, repo, workflowFile string, report *models.PerformanceReport) error {
	var totalTime time.Duration

	runs, err := a.client.GetWorkflowRuns(ctx, owner, repo, workflowFile)
	if err != nil {
		return fmt.Errorf("failed to get workflow runs: %v", err)
	}

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
					if strings.Contains(strategy.Example, "%s") {
						updatedStrategy.Example = fmt.Sprintf(strategy.Example, latestVersion)
					} else {
						updatedStrategy.Example = strategy.Example
					}
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
		"java": {
			"mvn",
			"maven",
			"gradle",
			"setup-java",
			"actions/setup-java",
			"pom.xml",
			"build.gradle",
			"java-version",
			"uses: actions/setup-java",
		},
		"ruby": {
			"bundle",
			"gem",
			"setup-ruby",
			"ruby/setup-ruby",
			"Gemfile",
			"ruby-version",
			"uses: ruby/setup-ruby",
		},
		"rust": {
			"cargo",
			"rustc",
			"rust-toolchain",
			"dtolnay/rust-toolchain",
			"Cargo.toml",
			"uses: dtolnay/rust-toolchain",
		},
		"dotnet": {
			"dotnet",
			"setup-dotnet",
			"actions/setup-dotnet",
			".csproj",
			".sln",
			"nuget",
			"dotnet-version",
			"uses: actions/setup-dotnet",
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

// // TODO: This function will be used in future for escaping GitHub expressions in example strings
// // Currently unused but kept for future implementation
// func _escapeGitHubExpression(example string) string {
// 	return strings.ReplaceAll(example, "${{", "\\${\\{")
// }

// analyzeWorkflowStructure analyzes the workflow structure and patterns
func (a *Analyzer) analyzeWorkflowStructure(content string, report *models.PerformanceReport) error {
	// GitHub 표현식 이스케이프 처리 추가
	content = strings.ReplaceAll(content, "${", "\\${")

	analysis := &models.WorkflowAnalysis{
		Recommendations:     make([]string, 0),
		RunnerOptimizations: make([]string, 0),
		SecurityTips:        make([]string, 0),
	}

	// Check for matrix strategy
	if !strings.Contains(content, "strategy:") || !strings.Contains(content, "matrix:") {
		analysis.Recommendations = append(analysis.Recommendations,
			"Consider using matrix strategy for parallel testing/building across different versions/platforms")
	}

	// Check for job dependencies
	if strings.Contains(content, "needs:") {
		analysis.ParallelJobs = true
		analysis.Recommendations = append(analysis.Recommendations,
			"Review job dependencies to ensure optimal parallel execution")
	}

	// Analyze runners
	if strings.Contains(content, "runs-on: ubuntu-latest") {
		analysis.RunnerOptimizations = append(analysis.RunnerOptimizations,
			"Consider using specific Ubuntu version instead of 'latest' for better reproducibility")
	}

	// Security checks
	if !strings.Contains(content, "permissions:") {
		analysis.SecurityTips = append(analysis.SecurityTips,
			"Add explicit permissions to improve workflow security")
	}

	// Check environment usage
	if !strings.Contains(content, "environment:") {
		analysis.SecurityTips = append(analysis.SecurityTips,
			"Consider using environments for better secret management and deployment control")
	}

	report.WorkflowAnalysis = analysis
	return nil
}
