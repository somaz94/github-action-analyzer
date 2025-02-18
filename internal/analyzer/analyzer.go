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
	client         GithubClient
	versionChecker VersionChecker
}

type GithubClient interface {
	GetWorkflowRuns(ctx context.Context, owner, repo, workflowFile string) ([]*gh.WorkflowRun, error)
	GetWorkflowJobLogs(ctx context.Context, owner, repo string, runID int64) (string, error)
	GetFileContent(ctx context.Context, owner, repo, path string) (string, error)
	GetLatestRelease(ctx context.Context, owner, repo string) (*gh.RepositoryRelease, error)
}

type VersionChecker interface {
	GetLatestVersion(lang string) (string, error)
}

type GitHubVersionChecker struct {
	client GithubClient
}

func (g *GitHubVersionChecker) GetLatestVersion(lang string) (string, error) {
	ctx := context.Background()
	switch lang {
	case "go":
		// Go 버전은 golang/go 레포지토리의 최신 릴리스 태그 확인
		release, err := g.client.GetLatestRelease(ctx, "golang", "go")
		if err != nil {
			return "", err
		}
		// "go1.24.0" 형식에서 버전만 추출
		version := strings.TrimPrefix(release.GetTagName(), "go")
		return strings.Split(version, ".")[0] + "." + strings.Split(version, ".")[1], nil

	case "node":
		// Node.js 버전은 nodejs/node 레포지토리 확인
		release, err := g.client.GetLatestRelease(ctx, "nodejs", "node")
		if err != nil {
			return "", err
		}
		return strings.TrimPrefix(release.GetTagName(), "v"), nil

	case "python":
		// Python 버전은 python/cpython 레포지토리 확인
		release, err := g.client.GetLatestRelease(ctx, "python", "cpython")
		if err != nil {
			return "", err
		}
		return strings.TrimPrefix(release.GetTagName(), "v"), nil

	default:
		return "", fmt.Errorf("unsupported language: %s", lang)
	}
}

func NewAnalyzer(client GithubClient) *Analyzer {
	return &Analyzer{
		client:         client,
		versionChecker: &GitHubVersionChecker{client: client},
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
		"go":     "1.24",
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

// Modifying the analyzeCaching method of Analyzer structures as well
func (a *Analyzer) analyzeCaching(ctx context.Context, owner, repo string, runs []*gh.WorkflowRun, report *models.PerformanceReport) error {
	// 기본 GitHub Actions 캐시 추천사항 항상 추가
	report.CacheRecommendations = append(report.CacheRecommendations, actionsCacheStrategies...)

	// 워크플로우 파일 경로 조정
	workflowPath := report.WorkflowFile
	if !strings.HasPrefix(workflowPath, ".github/workflows/") {
		workflowPath = fmt.Sprintf(".github/workflows/%s", workflowPath)
	}

	// 워크플로우 파일 내용 가져오기
	workflowContent, err := a.client.GetFileContent(ctx, owner, repo, workflowPath)
	if err == nil { // 파일을 찾은 경우에만 언어 감지
		// 워크플로우 파일에서 사용된 언어 감지
		detectedLangs := detectLanguagesFromWorkflow(workflowContent)

		// 감지된 언어별로 최신 버전 확인 및 캐시 전략 추가
		for _, lang := range detectedLangs {
			latestVersion, err := a.versionChecker.GetLatestVersion(lang)
			if err != nil {
				// 버전 확인 실패 시 기본 전략만 추가
				if strategies, ok := cacheStrategies[lang]; ok {
					report.CacheRecommendations = append(report.CacheRecommendations, strategies...)
				}
				continue
			}

			// 캐시 전략에 최신 버전 정보 반영
			if strategies, ok := cacheStrategies[lang]; ok {
				for _, strategy := range strategies {
					strategy.Example = strings.ReplaceAll(strategy.Example,
						fmt.Sprintf("'%s'", latestVersions[lang]),
						fmt.Sprintf("'%s'", latestVersion))
					report.CacheRecommendations = append(report.CacheRecommendations, strategy)
				}
			}
		}
	}

	// 실행 이력이 있는 경우 추가 분석
	if len(runs) > 0 {
		for _, run := range runs {
			recommendations, err := analyzeCacheHitPatterns(ctx, owner, repo, run, a.client)
			if err != nil {
				continue // 개별 실행 분석 실패는 무시
			}
			report.CacheRecommendations = append(report.CacheRecommendations, recommendations...)
		}
	}

	// 중복 제거
	report.CacheRecommendations = deduplicateCacheRecommendations(report.CacheRecommendations)

	return nil
}

// 캐시 추천사항 중복 제거
func deduplicateCacheRecommendations(recommendations []models.CacheRecommendation) []models.CacheRecommendation {
	seen := make(map[string]bool)
	var result []models.CacheRecommendation

	for _, rec := range recommendations {
		key := rec.Path + rec.Description
		if !seen[key] {
			seen[key] = true
			result = append(result, rec)
		}
	}

	return result
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

// 워크플로우 파일에서 사용된 언어 감지
func detectLanguagesFromWorkflow(content string) []string {
	var languages []string
	languagePatterns := map[string][]string{
		"go":     {"go build", "go test", "setup-go", "actions/setup-go"},
		"node":   {"npm", "yarn", "setup-node", "actions/setup-node", "package.json"},
		"python": {"pip", "python", "setup-python", "actions/setup-python", "requirements.txt"},
	}

	for lang, patterns := range languagePatterns {
		for _, pattern := range patterns {
			if strings.Contains(content, pattern) {
				languages = append(languages, lang)
				break
			}
		}
	}

	return unique(languages)
}
