// Copyright 2020 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// WorkflowRun represents a repository action workflow run.
type WorkflowRun struct {
	ID             *int64         `json:"id,omitempty"`
	NodeID         *string        `json:"node_id,omitempty"`
	HeadBranch     *string        `json:"head_branch,omitempty"`
	HeadSHA        *string        `json:"head_sha,omitempty"`
	RunNumber      *int           `json:"run_number,omitempty"`
	Event          *string        `json:"event,omitempty"`
	Status         *string        `json:"status,omitempty"`
	Conclusion     *string        `json:"conclusion,omitempty"`
	URL            *string        `json:"url,omitempty"`
	HTMLURL        *string        `json:"html_url,omitempty"`
	PullRequests   []*PullRequest `json:"pull_requests,omitempty"`
	CreatedAt      *Timestamp     `json:"created_at,omitempty"`
	UpdatedAt      *Timestamp     `json:"updated_at,omitempty"`
	JobsURL        *string        `json:"jobs_url,omitempty"`
	LogsURL        *string        `json:"logs_url,omitempty"`
	CheckSuiteURL  *string        `json:"check_suite_url,omitempty"`
	ArtifactsURL   *string        `json:"artifacts_url,omitempty"`
	CancelURL      *string        `json:"cancel_url,omitempty"`
	RerunURL       *string        `json:"rerun_url,omitempty"`
	HeadCommit     *HeadCommit    `json:"head_commit,omitempty"`
	WorkflowURL    *string        `json:"workflow_url,omitempty"`
	Repository     *Repository    `json:"repository,omitempty"`
	HeadRepository *Repository    `json:"head_repository,omitempty"`
}

// WorkflowRuns represents a slice of repository action workflow run.
type WorkflowRuns struct {
	TotalCount   *int           `json:"total_count,omitempty"`
	WorkflowRuns []*WorkflowRun `json:"workflow_runs,omitempty"`
}

// ListWorkflowRunsOptions specifies optional parameters to ListWorkflowRuns.
type ListWorkflowRunsOptions struct {
	Actor  string `url:"actor,omitempty"`
	Branch string `url:"branch,omitempty"`
	Event  string `url:"event,omitempty"`
	Status string `url:"status,omitempty"`
	ListOptions
}

func (s *ActionsService) listWorkflowRuns(ctx context.Context, endpoint string, opts *ListWorkflowRunsOptions) (*WorkflowRuns, *Response, error) {
	u, err := addOptions(endpoint, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	runs := new(WorkflowRuns)
	resp, err := s.client.Do(ctx, req, &runs)
	if err != nil {
		return nil, resp, err
	}

	return runs, resp, nil
}

// ListWorkflowRunsByID lists all workflow runs by workflow ID.
//
// GitHub API docs: https://developer.github.com/v3/actions/workflow_runs/#list-workflow-runs
func (s *ActionsService) ListWorkflowRunsByID(ctx context.Context, owner, repo string, workflowID int64, opts *ListWorkflowRunsOptions) (*WorkflowRuns, *Response, error) {
	u := fmt.Sprintf("repos/%s/%s/actions/workflows/%v/runs", owner, repo, workflowID)
	return s.listWorkflowRuns(ctx, u, opts)
}

// ListWorkflowRunsByFileName lists all workflow runs by workflow file name.
//
// GitHub API docs: https://developer.github.com/v3/actions/workflow_runs/#list-workflow-runs
func (s *ActionsService) ListWorkflowRunsByFileName(ctx context.Context, owner, repo, workflowFileName string, opts *ListWorkflowRunsOptions) (*WorkflowRuns, *Response, error) {
	u := fmt.Sprintf("repos/%s/%s/actions/workflows/%v/runs", owner, repo, workflowFileName)
	return s.listWorkflowRuns(ctx, u, opts)
}

// ListRepositoryWorkflowRuns lists all workflow runs for a repository.
//
// GitHub API docs: https://developer.github.com/v3/actions/workflow_runs/#list-repository-workflow-runs
func (s *ActionsService) ListRepositoryWorkflowRuns(ctx context.Context, owner, repo string, opts *ListOptions) (*WorkflowRuns, *Response, error) {
	u := fmt.Sprintf("repos/%s/%s/actions/runs", owner, repo)
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	runs := new(WorkflowRuns)
	resp, err := s.client.Do(ctx, req, &runs)
	if err != nil {
		return nil, resp, err
	}

	return runs, resp, nil
}

// GetWorkflowRunByID gets a specific workflow run by ID.
//
// GitHub API docs: https://developer.github.com/v3/actions/workflow_runs/#get-a-workflow-run
func (s *ActionsService) GetWorkflowRunByID(ctx context.Context, owner, repo string, runID int64) (*WorkflowRun, *Response, error) {
	u := fmt.Sprintf("repos/%v/%v/actions/runs/%v", owner, repo, runID)

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	run := new(WorkflowRun)
	resp, err := s.client.Do(ctx, req, run)
	if err != nil {
		return nil, resp, err
	}

	return run, resp, nil
}

// RerunWorkflow re-runs a workflow by ID.
//
// GitHub API docs: https://developer.github.com/v3/actions/workflow_runs/#re-run-a-workflow
func (s *ActionsService) RerunWorkflowByID(ctx context.Context, owner, repo string, runID int64) (*Response, error) {
	u := fmt.Sprintf("repos/%v/%v/actions/runs/%v/rerun", owner, repo, runID)

	req, err := s.client.NewRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// CancelWorkflowRunByID cancels a workflow run by ID.
//
// GitHub API docs: https://developer.github.com/v3/actions/workflow_runs/#cancel-a-workflow-run
func (s *ActionsService) CancelWorkflowRunByID(ctx context.Context, owner, repo string, runID int64) (*Response, error) {
	u := fmt.Sprintf("repos/%v/%v/actions/runs/%v/cancel", owner, repo, runID)

	req, err := s.client.NewRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// GetWorkflowRunLogs gets a redirect URL to download a plain text file of logs for a workflow run.
//
// GitHub API docs: https://developer.github.com/v3/actions/workflow_runs/#list-workflow-run-logs
func (s *ActionsService) GetWorkflowRunLogs(ctx context.Context, owner, repo string, runID int64, followRedirects bool) (*url.URL, *Response, error) {
	u := fmt.Sprintf("repos/%v/%v/actions/runs/%v/logs", owner, repo, runID)

	resp, err := s.getWorkflowLogsFromURL(ctx, u, followRedirects)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusFound {
		return nil, newResponse(resp), fmt.Errorf("unexpected status code: %s", resp.Status)
	}
	parsedURL, err := url.Parse(resp.Header.Get("Location"))
	return parsedURL, newResponse(resp), err
}
