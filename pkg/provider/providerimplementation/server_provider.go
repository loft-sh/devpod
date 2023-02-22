package providerimplementation

import (
	"bytes"
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/provider/options"
	"github.com/loft-sh/devpod/pkg/shell"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
	agentConfig := s.config.Agent
	if agentConfig.Path == "" {
		agentConfig.Path = agent.RemoteDevPodHelperLocation
	}
	if agentConfig.DownloadURL == "" {
		agentConfig.DownloadURL = agent.DefaultAgentDownloadURL
	}

	return &agentConfig, nil
}

func (s *serverProvider) Init(ctx context.Context, workspace *provider.Workspace, options provider.InitOptions) error {
	return runProviderCommand(ctx, "init", s.config.Exec.Init, workspace, s, os.Stdin, os.Stdout, os.Stderr, nil, s.log)
}

func (s *serverProvider) Validate(ctx context.Context, workspace *provider.Workspace, options provider.ValidateOptions) error {
	return runProviderCommand(ctx, "validate", s.config.Exec.Validate, workspace, s, os.Stdin, os.Stdout, os.Stderr, nil, s.log)
}

func (s *serverProvider) Create(ctx context.Context, workspace *provider.Workspace, options provider.CreateOptions) error {
	// provider doesn't support servers
	if len(s.config.Exec.Create) == 0 {
		return nil
	}

	// create a new server
	if workspace.Server.ID == "" {
		workspace.Server.ID = workspace.ID
		workspace.Server.AutoDelete = true

		err := provider.SaveWorkspaceConfig(workspace)
		if err != nil {
			return err
		}
	} else {
		// check if server already exists
		_, err := provider.LoadServerConfig(workspace.Context, workspace.Server.ID)
		if err == nil {
			return nil
		}
	}

	// create a server
	s.log.Infof("Create %s server...", s.config.Name)
	err := runProviderCommand(ctx, "create", s.config.Exec.Create, workspace, s, os.Stdin, os.Stdout, os.Stderr, nil, s.log)
	if err != nil {
		return err
	}

	// create server folder
	err = provider.SaveServerConfig(&provider.Server{
		ID:      workspace.Server.ID,
		Context: workspace.Context,
		Provider: provider.ServerProviderConfig{
			Name:    workspace.Provider.Name,
			Options: workspace.Provider.Options,
		},
		CreationTimestamp: types.Now(),
	})
	if err != nil {
		return err
	}

	s.log.Donef("Successfully created %s server", s.config.Name)
	return nil
}

func (s *serverProvider) Delete(ctx context.Context, workspace *provider.Workspace, options provider.DeleteOptions) error {
	// should just delete container?
	if !workspace.Server.AutoDelete {
		writer := s.log.Writer(logrus.InfoLevel, false)
		defer writer.Close()

		s.log.Infof("Deleting container...")
		err := runProviderCommand(ctx, "command", s.config.Exec.Command, workspace, s, nil, writer, writer, map[string]string{
			provider.CommandEnv: fmt.Sprintf("%s agent workspace delete --id %s --context %s", workspace.Provider.Agent.Path, workspace.ID, workspace.Context),
		}, s.log.ErrorStreamOnly())
		if err != nil {
			if !options.Force {
				return err
			}

			s.log.Errorf("Error deleting container: %v", err)
		} else {
			s.log.Infof("Successfully deleted container...")
		}
	} else if workspace.Server.ID != "" && len(s.config.Exec.Delete) > 0 {
		s.log.Infof("Deleting %s server...", s.config.Name)
		err := runProviderCommand(ctx, "delete", s.config.Exec.Delete, workspace, s, os.Stdin, os.Stdout, os.Stderr, nil, s.log)
		if err != nil {
			if !options.Force {
				return err
			}

			s.log.Errorf("Error deleting workspace %s", workspace.ID)
		}
		s.log.Donef("Successfully deleted %s server", s.config.Name)

		// delete server folder
		err = DeleteServerFolder(workspace.Context, workspace.Server.ID)
		if err != nil {
			return err
		}
	}

	return DeleteWorkspaceFolder(workspace.Context, workspace.ID)
}

func (s *serverProvider) Start(ctx context.Context, workspace *provider.Workspace, options provider.StartOptions) error {
	if workspace.Server.ID == "" {
		return nil
	}

	err := runProviderCommand(ctx, "start", s.config.Exec.Start, workspace, s, os.Stdin, os.Stdout, os.Stderr, nil, s.log)
	if err != nil {
		return err
	}

	return nil
}

func (s *serverProvider) Stop(ctx context.Context, workspace *provider.Workspace, options provider.StopOptions) error {
	if !workspace.Server.AutoDelete {
		writer := s.log.Writer(logrus.InfoLevel, false)
		defer writer.Close()

		// TODO: stop whole machine if there is no other workspace container running anymore

		s.log.Infof("Stopping container...")
		err := runProviderCommand(ctx, "command", s.config.Exec.Command, workspace, s, nil, writer, writer, map[string]string{
			provider.CommandEnv: fmt.Sprintf("%s agent workspace stop --id %s --context %s", workspace.Provider.Agent.Path, workspace.ID, workspace.Context),
		}, s.log.ErrorStreamOnly())
		if err != nil {
			return err
		}
		s.log.Infof("Successfully stopped container...")

		return nil
	}

	err := runProviderCommand(ctx, "stop", s.config.Exec.Stop, workspace, s, os.Stdin, os.Stdout, os.Stderr, nil, s.log)
	if err != nil {
		return err
	}

	return nil
}

func (s *serverProvider) Command(ctx context.Context, workspace *provider.Workspace, options provider.CommandOptions) error {
	err := runProviderCommand(ctx, "command", s.config.Exec.Command, workspace, s, options.Stdin, options.Stdout, options.Stderr, map[string]string{
		provider.CommandEnv: options.Command,
	}, s.log.ErrorStreamOnly())
	if err != nil {
		return err
	}

	return nil
}

func (s *serverProvider) Status(ctx context.Context, workspace *provider.Workspace, options provider.StatusOptions) (provider.Status, error) {
	// check if provider has status command
	if workspace.Server.ID != "" && len(s.config.Exec.Status) > 0 {
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		err := runProviderCommand(ctx, "status", s.config.Exec.Status, workspace, s, nil, stdout, stderr, nil, s.log)
		if err != nil {
			return provider.StatusNotFound, fmt.Errorf("get status: %s%s", strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()))
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
	workspaceFolder, err := provider.GetWorkspaceDir(workspace.Context, workspace.ID)
	if err != nil {
		return "", err
	}

	// does workspace folder exist?
	_, err = os.Stat(workspaceFolder)
	if err == nil {
		return provider.StatusRunning, nil
	}

	return provider.StatusNotFound, nil
}

func runProviderCommand(ctx context.Context, name string, command types.StrArray, workspace *provider.Workspace, prov provider.Provider, stdin io.Reader, stdout io.Writer, stderr io.Writer, extraEnv map[string]string, log log.Logger) (err error) {
	if len(command) == 0 {
		return nil
	}

	// log
	log.Debugf("Run %s provider command: %s", name, strings.Join(command, " "))

	// resolve options
	if workspace != nil {
		workspace, err = options.ResolveAndSaveOptions(ctx, name, "", workspace, prov)
		if err != nil {
			return err
		}
		defer func() {
			if err == nil {
				_, err = options.ResolveAndSaveOptions(ctx, "", name, workspace, prov)
			}
		}()
	}

	// run the command
	return RunCommand(ctx, command, workspace, stdin, stdout, stderr, extraEnv)
}

func RunCommand(ctx context.Context, command types.StrArray, workspace *provider.Workspace, stdin io.Reader, stdout io.Writer, stderr io.Writer, extraEnv map[string]string) error {
	env, err := provider.ToEnvironment(workspace)
	if err != nil {
		return err
	}

	// create environment variables for command
	osEnviron := os.Environ()
	osEnviron = append(osEnviron, env...)
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
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func DeleteServerFolder(context, serverID string) error {
	serverDir, err := provider.GetServerDir(context, serverID)
	if err != nil {
		return err
	}

	// remove server folder
	err = os.RemoveAll(serverDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func DeleteWorkspaceFolder(context, workspaceID string) error {
	workspaceFolder, err := provider.GetWorkspaceDir(context, workspaceID)
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
