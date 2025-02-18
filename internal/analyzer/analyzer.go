package analyzer

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
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
	if err := a.analyzeCaching(ctx, owner, repo, runs, report); err != nil {
		return nil, err
	}

	// Generate cost saving tips
	a.generateCostSavingTips(report)

	return report, nil
}

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

var (
	// 최신 버전 정보
	latestVersions = map[string]string{
		"go":     "1.22",
		"node":   "20",
		"python": "3.12",
	}

	// GitHub Actions 관련 캐시 전략
	actionsCacheStrategies = []models.CacheRecommendation{
		{
			Path:        "~/.github/actions",
			Description: "Cache GitHub Actions",
			Impact:      "Can reduce workflow execution time by caching action downloads",
			Example: `      - name: Cache GitHub Actions
        uses: actions/cache@v4
        with:
          path: ~/.github/actions
          key: ${{ runner.os }}-actions-${{ hashFiles('.github/workflows/**') }}`,
		},
	}

	// 언어별 캐시 전략
	cacheStrategies = map[string][]models.CacheRecommendation{
		"go": {
			{
				Path:        "~/.cache/go-build",
				Description: "Cache Go build artifacts",
				Impact:      "Can reduce build time by up to 30%",
				Example: `      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '` + latestVersions["go"] + `'
          cache: true  # This enables Go build cache`,
			},
			{
				Path:        "~/go/pkg/mod",
				Description: "Cache Go modules",
				Impact:      "Can reduce dependency download time significantly",
				Example: `      - name: Cache Go modules
        uses: actions/cache@v4
        with:
          path: ~/go/pkg/mod
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
          node-version: '` + latestVersions["node"] + `'
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
          python-version: '` + latestVersions["python"] + `'
          cache: 'pip'  # This enables pip cache
          cache-dependency-path: |
            **/requirements.txt
            **/requirements-dev.txt`,
			},
		},
	}
)

// 워크플로우에서 사용된 언어/프레임워크 감지
func detectLanguages(logs string) []string {
	var languages []string
	languagePatterns := map[string][]string{
		"go":     {"go build", "go test", "go.mod", "go.sum"},
		"node":   {"npm", "yarn", "package.json", "node_modules"},
		"python": {"pip", "requirements.txt", "setup.py", "poetry"},
	}

	for lang, patterns := range languagePatterns {
		for _, pattern := range patterns {
			if strings.Contains(logs, pattern) {
				languages = append(languages, lang)
				break
			}
		}
	}

	return unique(languages)
}

// 중복 제거 헬퍼 함수
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

// 예시 문자열에서 GitHub Actions 표현식을 이스케이프
func escapeGitHubExpression(example string) string {
	return strings.ReplaceAll(example, "${{", "\\${\\{")
}

func analyzeCacheHitPatterns(ctx context.Context, owner, repo string, run *gh.WorkflowRun, client GithubClient) ([]models.CacheRecommendation, error) {
	uniqueRecommendations := make(map[string]models.CacheRecommendation)

	// GitHub Actions 캐시 추천 추가
	for _, rec := range actionsCacheStrategies {
		uniqueRecommendations[rec.Path] = rec
	}

	// 워크플로우 로그 가져오기
	logs, err := client.GetWorkflowJobLogs(ctx, owner, repo, run.GetID())
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow logs: %v", err)
	}

	// 사용된 언어 감지 및 버전 체크
	detectedLangs := detectLanguages(logs)
	for _, lang := range detectedLangs {
		// 언어별 캐시 전략 추가
		if strategies, ok := cacheStrategies[lang]; ok {
			for _, rec := range strategies {
				uniqueRecommendations[rec.Path] = rec
			}
		}

		// 버전 체크 및 업데이트 추천
		if outdated := detectOutdatedVersion(logs, lang); outdated {
			rec := models.CacheRecommendation{
				Path:        fmt.Sprintf("%s-version", lang),
				Description: fmt.Sprintf("Update %s to latest version %s", lang, latestVersions[lang]),
				Impact:      "Latest version includes performance improvements and security fixes",
				Example:     generateVersionUpdateExample(lang),
			}
			uniqueRecommendations[rec.Path] = rec
		}
	}

	var recommendations []models.CacheRecommendation
	for _, rec := range uniqueRecommendations {
		recommendations = append(recommendations, rec)
	}

	// 예시 문자열 이스케이프 처리
	for i := range recommendations {
		recommendations[i].Example = escapeGitHubExpression(recommendations[i].Example)
	}

	return recommendations, nil
}

func detectOutdatedVersion(logs, lang string) bool {
	// 버전 패턴 정의
	versionPatterns := map[string]string{
		"go":     `go version go([\d\.]+)`,
		"node":   `node v?([\d\.]+)`,
		"python": `python-?([\d\.]+)`,
	}

	if pattern, ok := versionPatterns[lang]; ok {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(logs)

		if len(matches) > 1 {
			currentVersion := matches[1]
			latestVersion := latestVersions[lang]

			// 버전 비교
			current := strings.Split(currentVersion, ".")
			latest := strings.Split(latestVersion, ".")

			// 메이저 버전 비교
			if len(current) > 0 && len(latest) > 0 {
				currentMajor, err1 := strconv.Atoi(current[0])
				latestMajor, err2 := strconv.Atoi(latest[0])

				if err1 == nil && err2 == nil && currentMajor < latestMajor {
					return true
				}
			}
		}
	}

	return false
}

func generateVersionUpdateExample(lang string) string {
	switch lang {
	case "go":
		return fmt.Sprintf(`      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '%s'`, latestVersions[lang])
	case "node":
		return fmt.Sprintf(`      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '%s'`, latestVersions[lang])
	case "python":
		return fmt.Sprintf(`      - name: Set up Python
        uses: actions/setup-python@v5
        with:
          python-version: '%s'`, latestVersions[lang])
	default:
		return ""
	}
}

// Analyzer 구조체의 analyzeCaching 메서드도 수정
func (a *Analyzer) analyzeCaching(ctx context.Context, owner, repo string, runs []*gh.WorkflowRun, report *models.PerformanceReport) error {
	// 중복 제거를 위한 맵
	uniqueRecommendations := make(map[string]models.CacheRecommendation)

	for _, run := range runs {
		recommendations, err := analyzeCacheHitPatterns(ctx, owner, repo, run, a.client)
		if err != nil {
			return err
		}

		// 중복 제거하면서 추가
		for _, rec := range recommendations {
			uniqueRecommendations[rec.Path+rec.Description] = rec
		}
	}

	// 중복 제거된 추천사항만 추가
	for _, rec := range uniqueRecommendations {
		report.CacheRecommendations = append(report.CacheRecommendations, rec)
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
