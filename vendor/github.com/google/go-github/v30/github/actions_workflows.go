// Copyright 2020 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github

import (
	"context"
	"fmt"
)

// Workflow represents a repository action workflow.
type Workflow struct {
	ID        *int64     `json:"id,omitempty"`
	NodeID    *string    `json:"node_id,omitempty"`
	Name      *string    `json:"name,omitempty"`
	Path      *string    `json:"path,omitempty"`
	State     *string    `json:"state,omitempty"`
	CreatedAt *Timestamp `json:"created_at,omitempty"`
	UpdatedAt *Timestamp `json:"updated_at,omitempty"`
	URL       *string    `json:"url,omitempty"`
	HTMLURL   *string    `json:"html_url,omitempty"`
	BadgeURL  *string    `json:"badge_url,omitempty"`
}

// Workflows represents a slice of repository action workflows.
type Workflows struct {
	TotalCount *int        `json:"total_count,omitempty"`
	Workflows  []*Workflow `json:"workflows,omitempty"`
}

// ListWorkflows lists all workflows in a repository.
//
// GitHub API docs: https://developer.github.com/v3/actions/workflows/#list-repository-workflows
func (s *ActionsService) ListWorkflows(ctx context.Context, owner, repo string, opts *ListOptions) (*Workflows, *Response, error) {
	u := fmt.Sprintf("repos/%s/%s/actions/workflows", owner, repo)
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	workflows := new(Workflows)
	resp, err := s.client.Do(ctx, req, &workflows)
	if err != nil {
		return nil, resp, err
	}

	return workflows, resp, nil
}

// GetWorkflowByID gets a specific workflow by ID.
//
// GitHub API docs: https://developer.github.com/v3/actions/workflows/#get-a-workflow
func (s *ActionsService) GetWorkflowByID(ctx context.Context, owner, repo string, workflowID int64) (*Workflow, *Response, error) {
	u := fmt.Sprintf("repos/%v/%v/actions/workflows/%v", owner, repo, workflowID)

	return s.getWorkflow(ctx, u)
}

// GetWorkflowByFileName gets a specific workflow by file name.
//
// GitHub API docs: https://developer.github.com/v3/actions/workflows/#get-a-workflow
func (s *ActionsService) GetWorkflowByFileName(ctx context.Context, owner, repo, workflowFileName string) (*Workflow, *Response, error) {
	u := fmt.Sprintf("repos/%v/%v/actions/workflows/%v", owner, repo, workflowFileName)

	return s.getWorkflow(ctx, u)
}

func (s *ActionsService) getWorkflow(ctx context.Context, url string) (*Workflow, *Response, error) {
	req, err := s.client.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	workflow := new(Workflow)
	resp, err := s.client.Do(ctx, req, workflow)
	if err != nil {
		return nil, resp, err
	}

	return workflow, resp, nil
}
