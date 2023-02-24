package clientimplementation

import (
	"bytes"
	"context"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"os"
)

func NewDirectClient(provider *provider.ProviderConfig, workspace *provider.Workspace, log log.Logger) client.WorkspaceClient {
	return &directClient{
		config:    provider,
		workspace: workspace,
		log:       log,
	}
}

type directClient struct {
	config    *provider.ProviderConfig
	workspace *provider.Workspace
	log       log.Logger
}

func (s *directClient) Provider() string {
	return s.config.Name
}

func (s *directClient) ProviderType() provider.ProviderType {
	return s.config.Type
}

func (s *directClient) Context() string {
	return s.workspace.Context
}

func (s *directClient) Workspace() string {
	return s.workspace.ID
}

func (s *directClient) WorkspaceConfig() *provider.Workspace {
	return provider.CloneWorkspace(s.workspace)
}

func (s *directClient) Options() map[string]*provider.ProviderOption {
	return s.config.Options
}

func (s *directClient) Create(ctx context.Context, options client.CreateOptions) error {
	return runCommand(ctx, "create", s.config.Exec.Create, ToEnvironment(s.workspace, nil), os.Stdin, os.Stdout, os.Stderr, s.log)
}

func (s *directClient) Delete(ctx context.Context, options client.DeleteOptions) error {
	// kill the command after the grace period
	if options.GracePeriod != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *options.GracePeriod)
		defer cancel()
	}

	err := runCommand(ctx, "delete", s.config.Exec.Delete, ToEnvironment(s.workspace, nil), os.Stdin, os.Stdout, os.Stderr, s.log)
	if err != nil {
		if !options.Force {
			return err
		}

		s.log.Errorf("Error deleting workspace %s", s.workspace.ID)
	}

	return DeleteWorkspaceFolder(s.workspace.Context, s.workspace.ID)
}

func (s *directClient) Start(ctx context.Context, options client.StartOptions) error {
	err := runCommand(ctx, "start", s.config.Exec.Start, ToEnvironment(s.workspace, nil), os.Stdin, os.Stdout, os.Stderr, s.log)
	if err != nil {
		return err
	}

	return nil
}

func (s *directClient) Stop(ctx context.Context, options client.StopOptions) error {
	err := runCommand(ctx, "stop", s.config.Exec.Stop, ToEnvironment(s.workspace, nil), os.Stdin, os.Stdout, os.Stderr, s.log)
	if err != nil {
		return err
	}

	return nil
}

func (s *directClient) Command(ctx context.Context, options client.CommandOptions) error {
	err := runCommand(ctx, "command", s.config.Exec.Command, ToEnvironment(s.workspace, nil), options.Stdin, options.Stdout, options.Stderr, s.log.ErrorStreamOnly())
	if err != nil {
		return err
	}

	return nil
}

func (s *directClient) Status(ctx context.Context, options client.StatusOptions) (client.Status, error) {
	// check if provider has status command
	if len(s.config.Exec.Status) > 0 {
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		err := runCommand(ctx, "status", s.config.Exec.Status, ToEnvironment(s.workspace, nil), nil, stdout, stderr, s.log)
		if err != nil {
			return client.StatusNotFound, errors.Wrapf(err, "get status: %s%s", stdout, stderr)
		}

		// parse status
		parsedStatus, err := client.ParseStatus(stdout.String())
		if err != nil {
			return client.StatusNotFound, err
		}

		return parsedStatus, nil
	}

	// logic:
	// - if workspace folder exists -> Running
	// - if workspace folder doesn't exist -> NotFound
	workspaceFolder, err := provider.GetWorkspaceDir(s.workspace.Context, s.workspace.ID)
	if err != nil {
		return "", err
	}

	// does workspace folder exist?
	_, err = os.Stat(workspaceFolder)
	if err != nil {
		return client.StatusRunning, nil
	}

	return client.StatusNotFound, nil
}
