package github

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	gh "github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
)

type Client struct {
	client *gh.Client
	ctx    context.Context
}

func NewClient(token string) *Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &Client{
		client: gh.NewClient(tc),
		ctx:    ctx,
	}
}

func (c *Client) GetWorkflowRuns(ctx context.Context, owner, repo, workflowFile string) ([]*gh.WorkflowRun, error) {
	var allRuns []*gh.WorkflowRun
	opts := &gh.ListWorkflowRunsOptions{
		ListOptions: gh.ListOptions{
			PerPage: 100,
		},
	}

	// Add retry logic
	for retries := 3; retries > 0; retries-- {
		runs, _, err := c.client.Actions.ListWorkflowRunsByFileName(ctx, owner, repo, workflowFile, opts)
		if err == nil {
			if runs != nil && runs.WorkflowRuns != nil {
				allRuns = append(allRuns, runs.WorkflowRuns...)
			}
			break
		}
		time.Sleep(time.Second * 2)
	}

	// 실행 기록이 없어도 빈 슬라이스 반환
	return allRuns, nil
}

func (c *Client) GetWorkflowJobLogs(ctx context.Context, owner, repo string, runID int64) (string, error) {
	jobs, _, err := c.client.Actions.ListWorkflowJobs(ctx, owner, repo, runID, &gh.ListWorkflowJobsOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list workflow jobs: %v", err)
	}

	var logs string
	for _, job := range jobs.Jobs {
		// Get raw logs URL
		rawLogsURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/jobs/%d/logs", owner, repo, job.GetID())

		req, err := http.NewRequestWithContext(ctx, "GET", rawLogsURL, nil)
		if err != nil {
			continue
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		logContent, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			continue
		}
		logs += string(logContent)
	}

	return logs, nil
}

func (c *Client) GetFileContent(ctx context.Context, owner, repo, path string) (string, error) {
	fileContent, _, _, err := c.client.Repositories.GetContents(ctx, owner, repo, path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get file content: %v", err)
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return "", fmt.Errorf("failed to decode content: %v", err)
	}

	return content, nil
}

func (c *Client) GetLatestRelease(ctx context.Context, owner, repo string) (*gh.RepositoryRelease, error) {
	release, _, err := c.client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release for %s/%s: %v", owner, repo, err)
	}
	return release, nil
}
