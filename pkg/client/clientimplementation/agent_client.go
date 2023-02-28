package clientimplementation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/options"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/shell"
	"github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"sync"
)

func NewAgentClient(devPodConfig *config.Config, prov *provider.ProviderConfig, workspace *provider.Workspace, log log.Logger) (client.AgentClient, error) {
	var serverConfig *provider.Server
	if workspace.Server.ID != "" {
		var err error
		serverConfig, err = provider.LoadServerConfig(workspace.Context, workspace.Server.ID)
		if err != nil {
			return nil, errors.Wrap(err, "load server config")
		}
	}

	return &agentClient{
		devPodConfig: devPodConfig,
		config:       prov,
		workspace:    workspace,
		server:       serverConfig,
		log:          log,
	}, nil
}

type agentClient struct {
	m sync.Mutex

	devPodConfig *config.Config
	config       *provider.ProviderConfig
	workspace    *provider.Workspace
	server       *provider.Server
	log          log.Logger
}

func (s *agentClient) Provider() string {
	return s.config.Name
}

func (s *agentClient) ProviderType() provider.ProviderType {
	return s.config.Type
}

func (s *agentClient) Workspace() string {
	return s.workspace.ID
}

func (s *agentClient) WorkspaceConfig() *provider.Workspace {
	return provider.CloneWorkspace(s.workspace)
}

func (s *agentClient) Server() string {
	if s.server != nil {
		return s.server.ID
	}

	return ""
}

func (s *agentClient) AgentPath() string {
	s.m.Lock()
	defer s.m.Unlock()

	return options.ResolveAgentConfig(s.devPodConfig, s.config).Path
}

func (s *agentClient) AgentURL() string {
	s.m.Lock()
	defer s.m.Unlock()

	return options.ResolveAgentConfig(s.devPodConfig, s.config).DownloadURL
}

func (s *agentClient) Context() string {
	return s.workspace.Context
}

func (s *agentClient) RefreshOptions(ctx context.Context, beforeStage, afterStage string) error {
	s.m.Lock()
	defer s.m.Unlock()

	var err error
	s.devPodConfig, err = options.ResolveAndSaveOptions(ctx, beforeStage, afterStage, s.devPodConfig, s.config)
	if err != nil {
		return err
	}

	return nil
}

func (s *agentClient) AgentConfig() provider.ProviderAgentConfig {
	s.m.Lock()
	defer s.m.Unlock()

	return options.ResolveAgentConfig(s.devPodConfig, s.config)
}

func (s *agentClient) AgentInfo() (string, error) {
	s.m.Lock()
	defer s.m.Unlock()

	// marshal config
	out, err := json.Marshal(&provider.AgentWorkspaceInfo{
		Workspace: s.workspace,
		Server:    s.server,
		Agent:     options.ResolveAgentConfig(s.devPodConfig, s.config),
		Options:   s.devPodConfig.ProviderOptions(s.Provider()),
	})
	if err != nil {
		return "", err
	}

	return compress.Compress(string(out))
}

func (s *agentClient) Create(ctx context.Context, options client.CreateOptions) error {
	s.m.Lock()
	defer s.m.Unlock()

	// provider doesn't support servers
	if !s.isServerProvider() {
		return nil
	}

	// create a new server
	if s.workspace.Server.ID != "" {
		return nil
	}

	// create a new server
	s.workspace = provider.CloneWorkspace(s.workspace)
	s.workspace.Server.ID = s.workspace.ID
	s.workspace.Server.AutoDelete = true

	// get the server dir
	serverDir, err := provider.GetServerDir(s.workspace.Context, s.workspace.Server.ID)
	if err != nil {
		return err
	}

	// save server config
	s.server = &provider.Server{
		ID:      s.workspace.Server.ID,
		Folder:  serverDir,
		Context: s.workspace.Context,
		Provider: provider.ServerProviderConfig{
			Name: s.workspace.Provider.Name,
		},
		CreationTimestamp: types.Now(),
	}

	// create server folder
	err = provider.SaveServerConfig(s.server)
	if err != nil {
		return err
	}

	// create server ssh keys
	_, err = ssh.GetPublicKeyBase(s.server.Folder)
	if err != nil {
		return err
	}

	// save workspace config
	err = provider.SaveWorkspaceConfig(s.workspace)
	if err != nil {
		return err
	}

	// create server
	return NewServerClient(s.devPodConfig, s.config, s.server, s.log).Create(ctx, options)
}

func (s *agentClient) Delete(ctx context.Context, opt client.DeleteOptions) error {
	s.m.Lock()
	defer s.m.Unlock()

	// kill the command after the grace period
	if opt.GracePeriod != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *opt.GracePeriod)
		defer cancel()
	}

	// should just delete container?
	if !s.isServerProvider() || !s.workspace.Server.AutoDelete {
		writer := s.log.Writer(logrus.InfoLevel, false)
		defer writer.Close()

		s.log.Infof("Deleting container...")
		command := fmt.Sprintf("%s agent workspace delete --id %s --context %s", options.ResolveAgentConfig(s.devPodConfig, s.config).Path, s.workspace.ID, s.workspace.Context)
		err := runCommand(ctx, "command", s.config.Exec.Command, ToEnvironment(s.workspace, s.server, s.devPodConfig.ProviderOptions(s.config.Name), map[string]string{
			provider.CommandEnv: command,
		}), nil, writer, writer, s.log.ErrorStreamOnly())
		if err != nil {
			if !opt.Force {
				return err
			}

			if err != context.DeadlineExceeded {
				s.log.Errorf("Error deleting container: %v", err)
			}
		}
	} else if s.workspace.Server.ID != "" && len(s.config.Exec.Delete) > 0 {
		// delete server if config was found
		err := NewServerClient(s.devPodConfig, s.config, s.server, s.log).Delete(ctx, opt)
		if err != nil {
			return err
		}
	}

	return DeleteWorkspaceFolder(s.workspace.Context, s.workspace.ID)
}

func (s *agentClient) Start(ctx context.Context, options client.StartOptions) error {
	s.m.Lock()
	defer s.m.Unlock()

	if !s.isServerProvider() {
		return nil
	}

	return NewServerClient(s.devPodConfig, s.config, s.server, s.log).Start(ctx, options)
}

func (s *agentClient) Stop(ctx context.Context, opt client.StopOptions) error {
	s.m.Lock()
	defer s.m.Unlock()

	if !s.isServerProvider() || !s.workspace.Server.AutoDelete {
		writer := s.log.Writer(logrus.InfoLevel, false)
		defer writer.Close()

		// TODO: stop whole machine if there is no other workspace container running anymore

		s.log.Infof("Stopping container...")
		command := fmt.Sprintf("%s agent workspace stop --id %s --context %s", options.ResolveAgentConfig(s.devPodConfig, s.config).Path, s.workspace.ID, s.workspace.Context)
		err := runCommand(ctx, "command", s.config.Exec.Command, ToEnvironment(s.workspace, s.server, s.devPodConfig.ProviderOptions(s.config.Name), map[string]string{
			provider.CommandEnv: command,
		}), nil, writer, writer, s.log.ErrorStreamOnly())
		if err != nil {
			return err
		}
		s.log.Infof("Successfully stopped container...")

		return nil
	}

	return NewServerClient(s.devPodConfig, s.config, s.server, s.log).Stop(ctx, opt)
}

func (s *agentClient) Command(ctx context.Context, commandOptions client.CommandOptions) (err error) {
	// resolve options
	if !commandOptions.SkipOptionsResolve {
		err := s.RefreshOptions(ctx, "command", "")
		if err != nil {
			return err
		}
	}

	// get environment variables
	s.m.Lock()
	environ := ToEnvironment(s.workspace, s.server, s.devPodConfig.ProviderOptions(s.config.Name), map[string]string{
		provider.CommandEnv: commandOptions.Command,
	})
	s.m.Unlock()

	// resolve options
	return runCommand(ctx, "command", s.config.Exec.Command, environ, commandOptions.Stdin, commandOptions.Stdout, commandOptions.Stderr, s.log.ErrorStreamOnly())
}

func (s *agentClient) Status(ctx context.Context, options client.StatusOptions) (client.Status, error) {
	s.m.Lock()
	defer s.m.Unlock()

	// check if provider has status command
	if s.isServerProvider() {
		if s.server == nil {
			return client.StatusNotFound, nil
		}

		status, err := NewServerClient(s.devPodConfig, s.config, s.server, s.log).Status(ctx, options)
		if err != nil {
			return status, err
		}

		// try to check container status and if that fails check workspace folder
		if status == client.StatusRunning && options.ContainerStatus {
			return s.getContainerStatus(ctx)
		}

		return status, err
	}

	// try to check container status and if that fails check workspace folder
	if options.ContainerStatus {
		return s.getContainerStatus(ctx)
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
	if err == nil {
		return client.StatusRunning, nil
	}

	return client.StatusNotFound, nil
}

func (s *agentClient) getContainerStatus(ctx context.Context) (client.Status, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := fmt.Sprintf("%s agent workspace status --id %s --context %s", options.ResolveAgentConfig(s.devPodConfig, s.config).Path, s.workspace.ID, s.workspace.Context)
	err := runCommand(ctx, "command", s.config.Exec.Command, ToEnvironment(s.workspace, s.server, s.devPodConfig.ProviderOptions(s.config.Name), map[string]string{
		provider.CommandEnv: command,
	}), nil, stdout, stderr, s.log.ErrorStreamOnly())
	if err != nil {
		return client.StatusNotFound, err
	}

	parsed, err := client.ParseStatus(stdout.String())
	if err != nil {
		return client.StatusNotFound, fmt.Errorf("error parsing container status: %s%s%v", stdout.String(), stderr.String(), err)
	}

	return parsed, nil
}

func (s *agentClient) isServerProvider() bool {
	return len(s.config.Exec.Create) > 0
}

func ToEnvironment(workspace *provider.Workspace, server *provider.Server, options map[string]config.OptionValue, extraEnv map[string]string) []string {
	env := provider.ToOptions(workspace, server, options)

	// create environment variables for command
	osEnviron := os.Environ()
	for k, v := range env {
		osEnviron = append(osEnviron, k+"="+v)
	}
	for k, v := range extraEnv {
		osEnviron = append(osEnviron, k+"="+v)
	}

	return osEnviron
}

func RunCommand(ctx context.Context, command types.StrArray, environ []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	if len(command) == 0 {
		return nil
	}

	// use shell if command length is equal 1
	if len(command) == 1 {
		return shell.ExecuteCommandWithShell(ctx, command[0], stdin, stdout, stderr, environ)
	}

	// run command
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = environ
	err := cmd.Run()
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
