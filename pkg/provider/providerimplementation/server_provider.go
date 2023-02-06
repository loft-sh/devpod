package providerimplementation

import (
	"bytes"
	"context"
	"fmt"
	config "github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/json"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/pkg/errors"
	"io"
	"os"
	"os/exec"
)

func NewServerProvider(provider *provider.ProviderConfig) provider.ServerProvider {
	return &serverProvider{
		config: provider,
	}
}

type serverProvider struct {
	config *provider.ProviderConfig
}

func (s *serverProvider) Name() string {
	return s.config.Name
}

func (s *serverProvider) Description() string {
	return s.config.Description
}

func (s *serverProvider) Options() map[string]*provider.ProviderOption {
	return s.config.Options
}

func (s *serverProvider) validate(workspace *provider.Workspace) error {
	if workspace.Provider.Name != s.config.Name {
		return fmt.Errorf("provider mismatch between existing workspace and new workspace: %s (existing) != %s (current)", workspace.Provider.Name, s.config.Name)
	}

	return nil
}

func (s *serverProvider) Init(ctx context.Context, workspace *provider.Workspace, options provider.InitOptions) error {
	err := s.validate(workspace)
	if err != nil {
		return err
	}

	return runProviderCommand(ctx, s.config.Exec.Init, workspace, os.Stdin, os.Stdout, os.Stderr, nil)
}

func (s *serverProvider) Create(ctx context.Context, workspace *provider.Workspace, options provider.CreateOptions) error {
	err := s.validate(workspace)
	if err != nil {
		return err
	}

	err = createWorkspaceFolder(workspace, s.Name())
	if err != nil {
		return err
	}

	return runProviderCommand(ctx, s.config.Exec.Create, workspace, os.Stdin, os.Stdout, os.Stderr, nil)
}

func (s *serverProvider) Delete(ctx context.Context, workspace *provider.Workspace, options provider.DeleteOptions) error {
	err := s.validate(workspace)
	if err != nil {
		return err
	}

	err = runProviderCommand(ctx, s.config.Exec.Delete, workspace, os.Stdin, os.Stdout, os.Stderr, nil)
	if err != nil {
		return err
	}

	return deleteWorkspaceFolder(workspace.Context, workspace.ID)
}

func (s *serverProvider) Start(ctx context.Context, workspace *provider.Workspace, options provider.StartOptions) error {
	err := s.validate(workspace)
	if err != nil {
		return err
	}

	err = runProviderCommand(ctx, s.config.Exec.Start, workspace, os.Stdin, os.Stdout, os.Stderr, nil)
	if err != nil {
		return err
	}

	return nil
}

func (s *serverProvider) Stop(ctx context.Context, workspace *provider.Workspace, options provider.StopOptions) error {
	err := s.validate(workspace)
	if err != nil {
		return err
	}

	err = runProviderCommand(ctx, s.config.Exec.Stop, workspace, os.Stdin, os.Stdout, os.Stderr, nil)
	if err != nil {
		return err
	}

	return nil
}

func (s *serverProvider) Command(ctx context.Context, workspace *provider.Workspace, options provider.CommandOptions) error {
	err := s.validate(workspace)
	if err != nil {
		return err
	}

	err = runProviderCommand(ctx, s.config.Exec.Command, workspace, options.Stdin, options.Stdout, options.Stderr, map[string]string{
		provider.CommandEnv: options.Command,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *serverProvider) Status(ctx context.Context, workspace *provider.Workspace, options provider.StatusOptions) (provider.Status, error) {
	err := s.validate(workspace)
	if err != nil {
		return "", err
	}

	// check if provider has status command
	if len(s.config.Exec.Status) > 0 {
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
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

func runProviderCommand(ctx context.Context, command types.StrArray, workspace *provider.Workspace, stdin io.Reader, stdout io.Writer, stderr io.Writer, extraEnv map[string]string) error {
	if len(command) == 0 {
		return nil
	}

	// use shell if command length is equal 1
	args := []string{}
	if len(command) == 1 {
		args = append(args, "sh", "-c")
	} else {
		// check if devpod is first arg
		if args[0] == "devpod" {
			devPodExec, err := os.Executable()
			if err != nil {
				return errors.Wrap(err, "get executable path")
			}

			args[0] = devPodExec
		}

	}
	args = append(args, command...)

	// create environment variables for command
	osEnviron := os.Environ()
	osEnviron = append(osEnviron, provider.ToEnvironment(workspace)...)
	for k, v := range extraEnv {
		osEnviron = append(osEnviron, k+"="+v)
	}

	// run command
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = osEnviron
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func createWorkspaceFolder(workspace *provider.Workspace, provider string) error {
	// save config
	workspace.CreationTimestamp = json.Now()
	workspace.Provider.Name = provider
	err := config.SaveWorkspaceConfig(workspace)
	if err != nil {
		return err
	}

	return nil
}

func deleteWorkspaceFolder(context, workspaceID string) error {
	workspaceFolder, err := config.GetWorkspaceDir(context, workspaceID)
	if err != nil {
		return err
	}

	// remove workspace folder
	err = os.RemoveAll(workspaceFolder)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
