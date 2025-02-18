package models

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type StepAnalysis struct {
	Name            string        `json:"name"`
	ExecutionTime   time.Duration `json:"execution_time"`
	IsSlowStep      bool          `json:"is_slow_step"`
	Recommendations []string      `json:"recommendations"`
}

type CacheRecommendation struct {
	Path        string `json:"path"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
}

type DockerOptimization struct {
	Issue       string `json:"issue"`
	Suggestion  string `json:"suggestion"`
	Improvement string `json:"improvement"`
}

type PerformanceReport struct {
	Repository           string                `json:"repository"`
	WorkflowFile         string                `json:"workflow_file"`
	TotalExecutionTime   time.Duration         `json:"total_execution_time"`
	SlowSteps            []StepAnalysis        `json:"slow_steps"`
	CacheRecommendations []CacheRecommendation `json:"cache_recommendations"`
	DockerOptimizations  []DockerOptimization  `json:"docker_optimizations"`
	CostSavingTips       []string              `json:"cost_saving_tips"`
}

func (r *PerformanceReport) Output() error {
	// Create summary output
	summary := fmt.Sprintf(`
📊 Workflow Analysis Report
Repository: %s
Workflow: %s
Total Execution Time: %v

🐌 Slow Steps:
`, r.Repository, r.WorkflowFile, r.TotalExecutionTime)

	for _, step := range r.SlowSteps {
		summary += fmt.Sprintf("- %s (%v)\n", step.Name, step.ExecutionTime)
		for _, rec := range step.Recommendations {
			summary += fmt.Sprintf("  ↳ %s\n", rec)
		}
	}

	summary += "\n🔄 Cache Recommendations:\n"
	for _, cache := range r.CacheRecommendations {
		summary += fmt.Sprintf("- %s: %s\n", cache.Path, cache.Description)
	}

	summary += "\n🐳 Docker Optimizations:\n"
	for _, docker := range r.DockerOptimizations {
		summary += fmt.Sprintf("- %s\n  Solution: %s\n", docker.Issue, docker.Suggestion)
	}

	summary += "\n💰 Cost Saving Tips:\n"
	for _, tip := range r.CostSavingTips {
		summary += fmt.Sprintf("- %s\n", tip)
	}

	// Write to GitHub Actions output
	fmt.Println(summary)

	// Create JSON report file
	jsonReport, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %v", err)
	}

	if err := os.WriteFile("workflow-analysis.json", jsonReport, 0644); err != nil {
		return fmt.Errorf("failed to write report file: %v", err)
	}

	return nil
}
