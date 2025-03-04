package models

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
	Example     string `json:"example"`
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
	WorkflowAnalysis     *WorkflowAnalysis     `json:"workflow_analysis"`
	Metrics              struct {
		AverageStepDuration time.Duration `json:"average_step_duration"`
		MaxStepDuration     time.Duration `json:"max_step_duration"`
		TotalSteps          int           `json:"total_steps"`
		FailedSteps         int           `json:"failed_steps"`
	} `json:"metrics"`
}

func (r *PerformanceReport) Output() error {
	r.calculateMetrics()

	summary := fmt.Sprintf(`
╭──────────────────────────────────────────────╮
│           Workflow Analysis Report            │
╰──────────────────────────────────────────────╯

📋 Overview
• Repository: %s
• Workflow: %s
• Total Execution Time: %v

`, r.Repository, r.WorkflowFile, r.TotalExecutionTime)

	if len(r.SlowSteps) > 0 {
		summary += "🐌 Slow Steps Detected\n"
		summary += "──────────────────────\n"
		for _, step := range r.SlowSteps {
			summary += fmt.Sprintf("  • %s (Duration: %v)\n", step.Name, step.ExecutionTime)
			for _, rec := range step.Recommendations {
				summary += fmt.Sprintf("    ↳ %s\n", rec)
			}
		}
		summary += "\n"
	}

	if len(r.CacheRecommendations) > 0 {
		summary += "🔄 Cache Optimization Tips\n"
		summary += "─────────────────────────\n"
		for _, cache := range r.CacheRecommendations {
			summary += fmt.Sprintf("  • %s\n", cache.Path)
			summary += fmt.Sprintf("    ↳ What: %s\n", cache.Description)
			summary += fmt.Sprintf("    ↳ Impact: %s\n", cache.Impact)
			if cache.Example != "" {
				summary += "    ↳ Example:\n"
				summary += fmt.Sprintf("      ```yaml\n%s\n      ```\n", cache.Example)
			}
			summary += "\n"
		}
	}

	if len(r.DockerOptimizations) > 0 {
		summary += "🐳 Docker Optimization Tips\n"
		summary += "──────────────────────────\n"
		for _, docker := range r.DockerOptimizations {
			summary += fmt.Sprintf("  • Issue: %s\n", docker.Issue)
			summary += fmt.Sprintf("    ↳ Solution: %s\n", docker.Suggestion)
			summary += fmt.Sprintf("    ↳ Expected Improvement: %s\n", docker.Improvement)
			summary += "\n"
		}
	}

	if len(r.CostSavingTips) > 0 {
		summary += "💰 Cost Saving Opportunities\n"
		summary += "──────────────────────────\n"
		for _, tip := range r.CostSavingTips {
			summary += fmt.Sprintf("  • %s\n", tip)
		}
		summary += "\n"
	}

	if r.WorkflowAnalysis != nil {
		summary += "⚙️ Workflow Structure Analysis\n"
		summary += "────────────────────────────\n"

		if len(r.WorkflowAnalysis.Recommendations) > 0 {
			summary += "  📝 General Recommendations:\n"
			for _, rec := range r.WorkflowAnalysis.Recommendations {
				summary += fmt.Sprintf("    • %s\n", rec)
			}
			summary += "\n"
		}

		if len(r.WorkflowAnalysis.RunnerOptimizations) > 0 {
			summary += "  🏃 Runner Optimizations:\n"
			for _, opt := range r.WorkflowAnalysis.RunnerOptimizations {
				summary += fmt.Sprintf("    • %s\n", opt)
			}
			summary += "\n"
		}

		if len(r.WorkflowAnalysis.SecurityTips) > 0 {
			summary += "  🔒 Security Recommendations:\n"
			for _, tip := range r.WorkflowAnalysis.SecurityTips {
				summary += fmt.Sprintf("    • %s\n", tip)
			}
			summary += "\n"
		}
	}

	summary += "╭──────────────────────────────────────────────╮\n"
	summary += "│            End of Analysis Report            │\n"
	summary += "╰──────────────────────────────────────────────╯\n"

	// Write to GitHub Actions output
	fmt.Println(summary)

	// Set GitHub Actions outputs
	if err := r.setGitHubOutputs(); err != nil {
		return fmt.Errorf("failed to set GitHub outputs: %v", err)
	}

	return nil
}

func (r *PerformanceReport) setGitHubOutputs() error {
	// Convert metrics to JSON
	metricsSummary, err := json.Marshal(r.Metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %v", err)
	}

	// Convert report sections to JSON strings
	performanceSummary, err := json.Marshal(map[string]interface{}{
		"repository":       r.Repository,
		"workflow_file":    r.WorkflowFile,
		"total_execution":  r.TotalExecutionTime.String(),
		"slow_steps_count": len(r.SlowSteps),
	})
	if err != nil {
		return err
	}

	// Escape GitHub expression in cache recommendations
	for i := range r.CacheRecommendations {
		r.CacheRecommendations[i].Example = strings.ReplaceAll(
			r.CacheRecommendations[i].Example,
			"${{",
			"${'{'}{",
		)
	}

	cacheRecs, err := json.Marshal(r.CacheRecommendations)
	if err != nil {
		return err
	}

	dockerOpts, err := json.Marshal(r.DockerOptimizations)
	if err != nil {
		return err
	}

	// Get GitHub output file path from environment
	outputFile := os.Getenv("GITHUB_OUTPUT")
	if outputFile == "" {
		return fmt.Errorf("GITHUB_OUTPUT environment variable not set")
	}

	// Open the file for appending
	f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open GITHUB_OUTPUT file: %v", err)
	}
	defer f.Close()

	// Write outputs to the file with proper escaping
	// Use delimiter to safely handle multiline values
	delimiter := "EOF_" + time.Now().Format("20060102150405")

	// Write each output with its own delimiter
	fmt.Fprintf(f, "metrics_summary<<%s\n%s\n%s\n", delimiter, metricsSummary, delimiter)
	fmt.Fprintf(f, "performance_summary<<%s\n%s\n%s\n", delimiter, performanceSummary, delimiter)
	fmt.Fprintf(f, "cache_recommendations<<%s\n%s\n%s\n", delimiter, cacheRecs, delimiter)
	fmt.Fprintf(f, "docker_optimizations<<%s\n%s\n%s\n", delimiter, dockerOpts, delimiter)
	fmt.Fprintf(f, "status=success\n")

	return nil
}

func (r *PerformanceReport) calculateMetrics() {
	var totalDuration time.Duration
	maxDuration := time.Duration(0)
	failedSteps := 0

	// SlowSteps가 비어있으면 기본값 설정
	if len(r.SlowSteps) == 0 {
		r.Metrics.AverageStepDuration = 0
		r.Metrics.MaxStepDuration = 0
		r.Metrics.TotalSteps = 0
		r.Metrics.FailedSteps = 0
		return
	}

	for _, step := range r.SlowSteps {
		totalDuration += step.ExecutionTime
		if step.ExecutionTime > maxDuration {
			maxDuration = step.ExecutionTime
		}
		if strings.Contains(strings.ToLower(step.Name), "failed") {
			failedSteps++
		}
	}

	r.Metrics.AverageStepDuration = totalDuration / time.Duration(len(r.SlowSteps))
	r.Metrics.MaxStepDuration = maxDuration
	r.Metrics.TotalSteps = len(r.SlowSteps)
	r.Metrics.FailedSteps = failedSteps
}
