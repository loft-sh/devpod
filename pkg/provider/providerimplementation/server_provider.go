package providerimplementation

import (
	"bytes"
	"context"
	"fmt"
	config "github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/shell"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/pkg/errors"
	"io"
	"os"
	"os/exec"
	"strings"
)

func NewServerProvider(provider *provider.ProviderConfig, log log.Logger) provider.ServerProvider {
	return &serverProvider{
		config: provider,
		log:    log,
	}
}

type serverProvider struct {
	config *provider.ProviderConfig
	log    log.Logger
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

func (s *serverProvider) AgentConfig() (*provider.ProviderAgentConfig, error) {
	// TODO: fill in options?
	return &s.config.Agent, nil
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

	logProviderCommand("init", s.config.Exec.Init, s.log)
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

	logProviderCommand("create", s.config.Exec.Create, s.log)
	return runProviderCommand(ctx, s.config.Exec.Create, workspace, os.Stdin, os.Stdout, os.Stderr, nil)
}

func (s *serverProvider) Delete(ctx context.Context, workspace *provider.Workspace, options provider.DeleteOptions) error {
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

func (s *serverProvider) Start(ctx context.Context, workspace *provider.Workspace, options provider.StartOptions) error {
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

func (s *serverProvider) Stop(ctx context.Context, workspace *provider.Workspace, options provider.StopOptions) error {
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

func (s *serverProvider) Command(ctx context.Context, workspace *provider.Workspace, options provider.CommandOptions) error {
	err := s.validate(workspace)
	if err != nil {
		return err
	}

	logProviderCommand("command", s.config.Exec.Command, s.log.ErrorStreamOnly())
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

func logProviderCommand(stage string, command types.StrArray, log log.Logger) {
	if len(command) == 0 {
		return
	}
	log.Debugf("Run %s provider command: %s", stage, strings.Join(command, " "))
}

func runProviderCommand(ctx context.Context, command types.StrArray, workspace *provider.Workspace, stdin io.Reader, stdout io.Writer, stderr io.Writer, extraEnv map[string]string) error {
	if len(command) == 0 {
		return nil
	}

	// create environment variables for command
	osEnviron := os.Environ()
	osEnviron = append(osEnviron, provider.ToEnvironment(workspace)...)
	for k, v := range extraEnv {
		osEnviron = append(osEnviron, k+"="+v)
	}

	// use shell if command length is equal 1
	if len(command) == 1 {
		return shell.ExecuteCommandWithShell(ctx, command[0], stdin, stdout, stderr, osEnviron)
	}

	// check if devpod is first arg
	if command[0] == "devpod" {
		devPodExec, err := os.Executable()
		if err != nil {
			return errors.Wrap(err, "get executable path")
		}

		command[0] = devPodExec
	}

	// run command
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
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
	workspace.CreationTimestamp = types.Now()
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
