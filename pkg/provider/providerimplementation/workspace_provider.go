package providerimplementation

import (
	"bytes"
	"context"
	"fmt"
	config "github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"os"
)

func NewWorkspaceProvider(provider *provider.ProviderConfig, log log.Logger) provider.WorkspaceProvider {
	return &workspaceProvider{
		config: provider,
		log:    log,
	}
}

type workspaceProvider struct {
	config *provider.ProviderConfig
	log    log.Logger
}

func (s *workspaceProvider) Name() string {
	return s.config.Name
}

func (s *workspaceProvider) Description() string {
	return s.config.Description
}

func (s *workspaceProvider) Options() map[string]*provider.ProviderOption {
	return s.config.Options
}

func (s *workspaceProvider) AgentConfig() (*provider.ProviderAgentConfig, error) {
	return nil, fmt.Errorf("agent config not supported in workspace providers")
}

func (s *workspaceProvider) validate(workspace *provider.Workspace) error {
	if workspace.Provider.Name != s.config.Name {
		return fmt.Errorf("provider mismatch between existing workspace and new workspace: %s (existing) != %s (current)", workspace.Provider.Name, s.config.Name)
	}

	return nil
}

func (s *workspaceProvider) Init(ctx context.Context, workspace *provider.Workspace, options provider.InitOptions) error {
	err := s.validate(workspace)
	if err != nil {
		return err
	}

	logProviderCommand("init", s.config.Exec.Status, s.log)
	return runProviderCommand(ctx, s.config.Exec.Init, workspace, os.Stdin, os.Stdout, os.Stderr, nil)
}

func (s *workspaceProvider) Create(ctx context.Context, workspace *provider.Workspace, options provider.WorkspaceCreateOptions) error {
	err := s.validate(workspace)
	if err != nil {
		return err
	}

	err = createWorkspaceFolder(workspace, s.Name())
	if err != nil {
		return err
	}

	logProviderCommand("create", s.config.Exec.Create, s.log)
	return runProviderCommand(ctx, s.config.Exec.Create, workspace, os.Stdin, os.Stdout, os.Stderr, nil)
}

func (s *workspaceProvider) Delete(ctx context.Context, workspace *provider.Workspace, options provider.WorkspaceDeleteOptions) error {
	err := s.validate(workspace)
	if err != nil {
		return err
	}

	logProviderCommand("delete", s.config.Exec.Delete, s.log)
	err = runProviderCommand(ctx, s.config.Exec.Delete, workspace, os.Stdin, os.Stdout, os.Stderr, nil)
	if err != nil {
		return err
	}

	return deleteWorkspaceFolder(workspace.Context, workspace.ID)
}

func (s *workspaceProvider) Start(ctx context.Context, workspace *provider.Workspace, options provider.WorkspaceStartOptions) error {
	err := s.validate(workspace)
	if err != nil {
		return err
	}

	logProviderCommand("start", s.config.Exec.Start, s.log)
	err = runProviderCommand(ctx, s.config.Exec.Start, workspace, os.Stdin, os.Stdout, os.Stderr, nil)
	if err != nil {
		return err
	}

	return nil
}

func (s *workspaceProvider) Stop(ctx context.Context, workspace *provider.Workspace, options provider.WorkspaceStopOptions) error {
	err := s.validate(workspace)
	if err != nil {
		return err
	}

	logProviderCommand("stop", s.config.Exec.Stop, s.log)
	err = runProviderCommand(ctx, s.config.Exec.Stop, workspace, os.Stdin, os.Stdout, os.Stderr, nil)
	if err != nil {
		return err
	}

	return nil
}

func (s *workspaceProvider) Tunnel(ctx context.Context, workspace *provider.Workspace, options provider.WorkspaceTunnelOptions) error {
	err := s.validate(workspace)
	if err != nil {
		return err
	}

	logProviderCommand("tunnel", s.config.Exec.Tunnel, s.log.ErrorStreamOnly())
	err = runProviderCommand(ctx, s.config.Exec.Tunnel, workspace, options.Stdin, options.Stdout, options.Stderr, nil)
	if err != nil {
		return err
	}

	return nil
}

func (s *workspaceProvider) Status(ctx context.Context, workspace *provider.Workspace, options provider.WorkspaceStatusOptions) (provider.Status, error) {
	err := s.validate(workspace)
	if err != nil {
		return "", err
	}

	// check if provider has status command
	if len(s.config.Exec.Status) > 0 {
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		logProviderCommand("status", s.config.Exec.Status, s.log)
		err := runProviderCommand(ctx, s.config.Exec.Status, workspace, nil, stdout, stderr, nil)
		if err != nil {
			return provider.StatusNotFound, errors.Wrapf(err, "get status: %s%s", stdout, stderr)
		}

		// parse status
		parsedStatus, err := provider.ParseStatus(stdout.String())
		if err != nil {
			return provider.StatusNotFound, err
		}

		return parsedStatus, nil
	}

	// logic:
	// - if workspace folder exists -> Running
	// - if workspace folder doesn't exist -> NotFound
	workspaceFolder, err := config.GetWorkspaceDir(workspace.Context, workspace.ID)
	if err != nil {
		return "", err
	}

	// does workspace folder exist?
	_, err = os.Stat(workspaceFolder)
	if err != nil {
		return provider.StatusRunning, nil
	}

	return provider.StatusNotFound, nil
}
