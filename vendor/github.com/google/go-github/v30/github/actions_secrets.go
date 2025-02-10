// Copyright 2020 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github

import (
	"context"
	"fmt"
)

// PublicKey represents the public key that should be used to encrypt secrets.
type PublicKey struct {
	KeyID *string `json:"key_id"`
	Key   *string `json:"key"`
}

// GetPublicKey gets a public key that should be used for secret encryption.
//
// GitHub API docs: https://developer.github.com/v3/actions/secrets/#get-your-public-key
func (s *ActionsService) GetPublicKey(ctx context.Context, owner, repo string) (*PublicKey, *Response, error) {
	u := fmt.Sprintf("repos/%v/%v/actions/secrets/public-key", owner, repo)
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	pubKey := new(PublicKey)
	resp, err := s.client.Do(ctx, req, pubKey)
	if err != nil {
		return nil, resp, err
	}

	return pubKey, resp, nil
}

// Secret represents a repository action secret.
type Secret struct {
	Name      string    `json:"name"`
	CreatedAt Timestamp `json:"created_at"`
	UpdatedAt Timestamp `json:"updated_at"`
}

// Secrets represents one item from the ListSecrets response.
type Secrets struct {
	TotalCount int       `json:"total_count"`
	Secrets    []*Secret `json:"secrets"`
}

// ListSecrets lists all secrets available in a repository
// without revealing their encrypted values.
//
// GitHub API docs: https://developer.github.com/v3/actions/secrets/#list-secrets-for-a-repository
func (s *ActionsService) ListSecrets(ctx context.Context, owner, repo string, opts *ListOptions) (*Secrets, *Response, error) {
	u := fmt.Sprintf("repos/%s/%s/actions/secrets", owner, repo)
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	secrets := new(Secrets)
	resp, err := s.client.Do(ctx, req, &secrets)
	if err != nil {
		return nil, resp, err
	}

	return secrets, resp, nil
}

// GetSecret gets a single secret without revealing its encrypted value.
//
// GitHub API docs: https://developer.github.com/v3/actions/secrets/#get-a-secret
func (s *ActionsService) GetSecret(ctx context.Context, owner, repo, name string) (*Secret, *Response, error) {
	u := fmt.Sprintf("repos/%v/%v/actions/secrets/%v", owner, repo, name)
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	secret := new(Secret)
	resp, err := s.client.Do(ctx, req, secret)
	if err != nil {
		return nil, resp, err
	}

	return secret, resp, nil
}

// EncryptedSecret represents a secret that is encrypted using a public key.
//
// The value of EncryptedValue must be your secret, encrypted with
// LibSodium (see documentation here: https://libsodium.gitbook.io/doc/bindings_for_other_languages)
// using the public key retrieved using the GetPublicKey method.
type EncryptedSecret struct {
	Name           string `json:"-"`
	KeyID          string `json:"key_id"`
	EncryptedValue string `json:"encrypted_value"`
}

// CreateOrUpdateSecret creates or updates a secret with an encrypted value.
//
// GitHub API docs: https://developer.github.com/v3/actions/secrets/#create-or-update-a-secret-for-a-repository
func (s *ActionsService) CreateOrUpdateSecret(ctx context.Context, owner, repo string, eSecret *EncryptedSecret) (*Response, error) {
	u := fmt.Sprintf("repos/%v/%v/actions/secrets/%v", owner, repo, eSecret.Name)

	req, err := s.client.NewRequest("PUT", u, eSecret)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// DeleteSecret deletes a secret in a repository using the secret name.
//
// GitHub API docs: https://developer.github.com/v3/actions/secrets/#delete-a-secret-from-a-repository
func (s *ActionsService) DeleteSecret(ctx context.Context, owner, repo, name string) (*Response, error) {
	u := fmt.Sprintf("repos/%v/%v/actions/secrets/%v", owner, repo, name)

	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
