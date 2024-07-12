// Copyright 2020 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github

import (
	"context"
	"fmt"
)

// RunnerApplicationDownload represents a binary for the self-hosted runner application that can be downloaded.
type RunnerApplicationDownload struct {
	OS           *string `json:"os,omitempty"`
	Architecture *string `json:"architecture,omitempty"`
	DownloadURL  *string `json:"download_url,omitempty"`
	Filename     *string `json:"filename,omitempty"`
}

// ListRunnerApplicationDownloads lists self-hosted runner application binaries that can be downloaded and run.
//
// GitHub API docs: https://developer.github.com/v3/actions/self_hosted_runners/#list-downloads-for-the-self-hosted-runner-application
func (s *ActionsService) ListRunnerApplicationDownloads(ctx context.Context, owner, repo string) ([]*RunnerApplicationDownload, *Response, error) {
	u := fmt.Sprintf("repos/%v/%v/actions/runners/downloads", owner, repo)
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var rads []*RunnerApplicationDownload
	resp, err := s.client.Do(ctx, req, &rads)
	if err != nil {
		return nil, resp, err
	}

	return rads, resp, nil
}

// RegistrationToken represents a token that can be used to add a self-hosted runner to a repository.
type RegistrationToken struct {
	Token     *string    `json:"token,omitempty"`
	ExpiresAt *Timestamp `json:"expires_at,omitempty"`
}

// CreateRegistrationToken creates a token that can be used to add a self-hosted runner.
//
// GitHub API docs: https://developer.github.com/v3/actions/self_hosted_runners/#create-a-registration-token
func (s *ActionsService) CreateRegistrationToken(ctx context.Context, owner, repo string) (*RegistrationToken, *Response, error) {
	u := fmt.Sprintf("repos/%v/%v/actions/runners/registration-token", owner, repo)

	req, err := s.client.NewRequest("POST", u, nil)
	if err != nil {
		return nil, nil, err
	}

	registrationToken := new(RegistrationToken)
	resp, err := s.client.Do(ctx, req, registrationToken)
	if err != nil {
		return nil, resp, err
	}

	return registrationToken, resp, nil
}

// Runner represents a self-hosted runner registered with a repository.
type Runner struct {
	ID     *int64  `json:"id,omitempty"`
	Name   *string `json:"name,omitempty"`
	OS     *string `json:"os,omitempty"`
	Status *string `json:"status,omitempty"`
}

// ListRunners lists all the self-hosted runners for a repository.
//
// GitHub API docs: https://developer.github.com/v3/actions/self_hosted_runners/#list-self-hosted-runners-for-a-repository
func (s *ActionsService) ListRunners(ctx context.Context, owner, repo string, opts *ListOptions) ([]*Runner, *Response, error) {
	u := fmt.Sprintf("repos/%v/%v/actions/runners", owner, repo)
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var runners []*Runner
	resp, err := s.client.Do(ctx, req, &runners)
	if err != nil {
		return nil, resp, err
	}

	return runners, resp, nil
}

// GetRunner gets a specific self-hosted runner for a repository using its runner ID.
//
// GitHub API docs: https://developer.github.com/v3/actions/self_hosted_runners/#get-a-self-hosted-runner
func (s *ActionsService) GetRunner(ctx context.Context, owner, repo string, runnerID int64) (*Runner, *Response, error) {
	u := fmt.Sprintf("repos/%v/%v/actions/runners/%v", owner, repo, runnerID)
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	runner := new(Runner)
	resp, err := s.client.Do(ctx, req, runner)
	if err != nil {
		return nil, resp, err
	}

	return runner, resp, nil
}

// RemoveToken represents a token that can be used to remove a self-hosted runner from a repository.
type RemoveToken struct {
	Token     *string    `json:"token,omitempty"`
	ExpiresAt *Timestamp `json:"expires_at,omitempty"`
}

// CreateRemoveToken creates a token that can be used to remove a self-hosted runner from a repository.
//
// GitHub API docs: https://developer.github.com/v3/actions/self_hosted_runners/#create-a-remove-token
func (s *ActionsService) CreateRemoveToken(ctx context.Context, owner, repo string) (*RemoveToken, *Response, error) {
	u := fmt.Sprintf("repos/%v/%v/actions/runners/remove-token", owner, repo)

	req, err := s.client.NewRequest("POST", u, nil)
	if err != nil {
		return nil, nil, err
	}

	removeToken := new(RemoveToken)
	resp, err := s.client.Do(ctx, req, removeToken)
	if err != nil {
		return nil, resp, err
	}

	return removeToken, resp, nil
}

// RemoveRunner forces the removal of a self-hosted runner in a repository using the runner id.
//
// GitHub API docs: https://developer.github.com/v3/actions/self_hosted_runners/#remove-a-self-hosted-runner
func (s *ActionsService) RemoveRunner(ctx context.Context, owner, repo string, runnerID int64) (*Response, error) {
	u := fmt.Sprintf("repos/%v/%v/actions/runners/%v", owner, repo, runnerID)

	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
