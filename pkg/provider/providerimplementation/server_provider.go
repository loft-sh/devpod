package providerimplementation

import (
	"bytes"
	"context"
	"fmt"
	config "github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	options2 "github.com/loft-sh/devpod/pkg/provider/options"
	"github.com/loft-sh/devpod/pkg/shell"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/pkg/errors"
	"io"
	"os"
	"os/exec"
	"reflect"
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
	return runProviderCommand(ctx, "init", s.config.Exec.Init, workspace, s.Options(), os.Stdin, os.Stdout, os.Stderr, nil, s.log)
}

func (s *serverProvider) Validate(ctx context.Context, workspace *provider.Workspace, options provider.ValidateOptions) error {
	return runProviderCommand(ctx, "validate", s.config.Exec.Validate, workspace, s.Options(), os.Stdin, os.Stdout, os.Stderr, nil, s.log)
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

	return runProviderCommand(ctx, "create", s.config.Exec.Create, workspace, s.Options(), os.Stdin, os.Stdout, os.Stderr, nil, s.log)
}

func (s *serverProvider) Delete(ctx context.Context, workspace *provider.Workspace, options provider.DeleteOptions) error {
	err := s.validate(workspace)
	if err != nil {
		return err
	}

	err = runProviderCommand(ctx, "delete", s.config.Exec.Delete, workspace, s.Options(), os.Stdin, os.Stdout, os.Stderr, nil, s.log)
	if err != nil {
		return err
	}

	return DeleteWorkspaceFolder(workspace.Context, workspace.ID)
}

func (s *serverProvider) Start(ctx context.Context, workspace *provider.Workspace, options provider.StartOptions) error {
	err := s.validate(workspace)
	if err != nil {
		return err
	}

	err = runProviderCommand(ctx, "start", s.config.Exec.Start, workspace, s.Options(), os.Stdin, os.Stdout, os.Stderr, nil, s.log)
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

	err = runProviderCommand(ctx, "stop", s.config.Exec.Stop, workspace, s.Options(), os.Stdin, os.Stdout, os.Stderr, nil, s.log)
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

	err = runProviderCommand(ctx, "command", s.config.Exec.Command, workspace, s.Options(), options.Stdin, options.Stdout, options.Stderr, map[string]string{
		provider.CommandEnv: options.Command,
	}, s.log.ErrorStreamOnly())
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
		err := runProviderCommand(ctx, "status", s.config.Exec.Status, workspace, s.Options(), nil, stdout, stderr, nil, s.log)
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

func runProviderCommand(ctx context.Context, name string, command types.StrArray, workspace *provider.Workspace, providerOptions map[string]*provider.ProviderOption, stdin io.Reader, stdout io.Writer, stderr io.Writer, extraEnv map[string]string, log log.Logger) (err error) {
	if len(command) == 0 {
		return nil
	}

	// log
	log.Debugf("Run %s provider command: %s", name, strings.Join(command, " "))

	// resolve options
	if workspace != nil {
		err = resolveOptions(ctx, name, "", workspace, providerOptions)
		if err != nil {
			return err
		}
		defer func() {
			if err == nil {
				err = resolveOptions(ctx, "", name, workspace, providerOptions)
			}
		}()
	}

	// run the command
	return RunCommand(ctx, command, workspace, stdin, stdout, stderr, extraEnv)
}

func RunCommand(ctx context.Context, command types.StrArray, workspace *provider.Workspace, stdin io.Reader, stdout io.Writer, stderr io.Writer, extraEnv map[string]string) error {
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

func resolveOptions(ctx context.Context, beforeStage, afterStage string, workspace *provider.Workspace, providerOptions map[string]*provider.ProviderOption) error {
	var err error

	// resolve options
	beforeOptions := workspace.Provider.Options
	workspace.Provider.Options, err = options2.ResolveOptions(ctx, beforeStage, afterStage, workspace, providerOptions)
	if err != nil {
		return errors.Wrap(err, "resolve options")
	}

	// save workspace config
	if workspace.ID != "" && !reflect.DeepEqual(workspace.Provider.Options, beforeOptions) {
		err = config.SaveWorkspaceConfig(workspace)
		if err != nil {
			return err
		}
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

func DeleteWorkspaceFolder(context, workspaceID string) error {
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
