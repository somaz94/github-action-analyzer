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
}

func (r *PerformanceReport) Output() error {
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

	// Get GITHUB_OUTPUT environment variable
	outputFile := os.Getenv("GITHUB_OUTPUT")
	if outputFile == "" {
		return fmt.Errorf("GITHUB_OUTPUT environment variable not set")
	}

	// Open the file in append mode
	f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open GITHUB_OUTPUT file: %v", err)
	}
	defer f.Close()

	// Write outputs to the file
	if _, err := fmt.Fprintf(f, "performance_summary=%s\n", performanceSummary); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(f, "cache_recommendations=%s\n", cacheRecs); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(f, "docker_optimizations=%s\n", dockerOpts); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(f, "status=success\n"); err != nil {
		return err
	}

	return nil
}
