package clientimplementation

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/provider/options"
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

func NewAgentClient(prov *provider.ProviderConfig, workspace *provider.Workspace, log log.Logger) (client.AgentClient, error) {
	var serverConfig *provider.Server
	if workspace.Server.ID != "" {
		var err error
		serverConfig, err = provider.LoadServerConfig(workspace.Context, workspace.Server.ID)
		if err != nil {
			return nil, errors.Wrap(err, "load server config")
		}
	}

	return &agentClient{
		config:    prov,
		workspace: workspace,
		server:    serverConfig,
		log:       log,
	}, nil
}

type agentClient struct {
	m sync.Mutex

	config    *provider.ProviderConfig
	workspace *provider.Workspace
	server    *provider.Server
	log       log.Logger
}

func (s *agentClient) Provider() string {
	return s.config.Name
}

func (s *agentClient) ProviderType() provider.ProviderType {
	return s.config.Type
}

func (s *agentClient) Workspace() string {
	s.m.Lock()
	defer s.m.Unlock()

	return s.workspace.ID
}

func (s *agentClient) WorkspaceConfig() *provider.Workspace {
	s.m.Lock()
	defer s.m.Unlock()

	return provider.CloneWorkspace(s.workspace)
}

func (s *agentClient) AgentPath() string {
	s.m.Lock()
	defer s.m.Unlock()

	return s.workspace.Provider.Agent.Path
}

func (s *agentClient) AgentURL() string {
	s.m.Lock()
	defer s.m.Unlock()

	return s.workspace.Provider.Agent.DownloadURL
}

func (s *agentClient) Context() string {
	s.m.Lock()
	defer s.m.Unlock()

	return s.workspace.Context
}

func (s *agentClient) Options() map[string]*provider.ProviderOption {
	s.m.Lock()
	defer s.m.Unlock()

	return s.config.Options
}

func (s *agentClient) RefreshOptions(beforeStage, afterStage string) error {
	s.m.Lock()
	defer s.m.Unlock()

	if s.workspace.Server.ID != "" {
		serverConfig, err := options.ResolveAndSaveOptionsServer(context.TODO(), beforeStage, afterStage, s.server, s.config)
		if err != nil {
			return err
		}

		s.server = serverConfig
		return nil
	}

	workspace, err := options.ResolveAndSaveOptions(context.TODO(), beforeStage, afterStage, s.workspace, s.config)
	if err != nil {
		return err
	}

	s.workspace = workspace
	return nil
}

func (s *agentClient) AgentInfo() (string, error) {
	s.m.Lock()
	defer s.m.Unlock()

	// trim options that don't exist
	workspace := provider.CloneWorkspace(s.workspace)
	if workspace.Server.ID != "" {
		newOptions := map[string]config.OptionValue{}
		for name, option := range s.server.Provider.Options {
			if option.Local {
				continue
			}

			newOptions[name] = option
		}
		workspace.Provider.Options = newOptions
	} else if workspace.Provider.Options != nil {
		for name, option := range workspace.Provider.Options {
			if option.Local {
				delete(workspace.Provider.Options, name)
			}
		}
	}

	// marshal config
	out, err := json.Marshal(&provider.AgentWorkspaceInfo{
		Workspace: *workspace,
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
	if len(s.config.Exec.Create) == 0 {
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
			Name:    s.workspace.Provider.Name,
			Options: s.workspace.Provider.Options,
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
	return NewServerClient(s.config, s.server, s.log).Create(ctx, options)
}

func (s *agentClient) Delete(ctx context.Context, options client.DeleteOptions) error {
	s.m.Lock()
	defer s.m.Unlock()

	// should just delete container?
	if s.workspace.Server.ID == "" || !s.workspace.Server.AutoDelete {
		writer := s.log.Writer(logrus.InfoLevel, false)
		defer writer.Close()

		s.log.Infof("Deleting container...")
		command := fmt.Sprintf("%s agent workspace delete --id %s --context %s", s.workspace.Provider.Agent.Path, s.workspace.ID, s.workspace.Context)
		var err error
		if s.workspace.Server.ID != "" {
			err = runCommand(ctx, "command", s.config.Exec.Command, ToEnvironmentServer(s.server, map[string]string{
				provider.CommandEnv: command,
			}), nil, writer, writer, s.log.ErrorStreamOnly())
		} else {
			err = runCommand(ctx, "command", s.config.Exec.Command, ToEnvironment(s.workspace, map[string]string{
				provider.CommandEnv: command,
			}), nil, writer, writer, s.log.ErrorStreamOnly())
		}
		if err != nil {
			if !options.Force {
				return err
			}

			s.log.Errorf("Error deleting container: %v", err)
		} else {
			s.log.Infof("Successfully deleted container...")
		}
	} else if s.workspace.Server.ID != "" && len(s.config.Exec.Delete) > 0 {
		// delete server if config was found
		err := NewServerClient(s.config, s.server, s.log).Delete(ctx, options)
		if err != nil {
			return err
		}
	}

	return DeleteWorkspaceFolder(s.workspace.Context, s.workspace.ID)
}

func (s *agentClient) Start(ctx context.Context, options client.StartOptions) error {
	s.m.Lock()
	defer s.m.Unlock()

	if s.workspace.Server.ID == "" {
		return nil
	}

	return NewServerClient(s.config, s.server, s.log).Start(ctx, options)
}

func (s *agentClient) Stop(ctx context.Context, options client.StopOptions) error {
	s.m.Lock()
	defer s.m.Unlock()

	if s.workspace.Server.ID == "" || !s.workspace.Server.AutoDelete {
		writer := s.log.Writer(logrus.InfoLevel, false)
		defer writer.Close()

		// TODO: stop whole machine if there is no other workspace container running anymore

		s.log.Infof("Stopping container...")
		command := fmt.Sprintf("%s agent workspace stop --id %s --context %s", s.workspace.Provider.Agent.Path, s.workspace.ID, s.workspace.Context)
		var err error
		if s.workspace.Server.ID != "" {
			err = runCommand(ctx, "command", s.config.Exec.Command, ToEnvironmentServer(s.server, map[string]string{
				provider.CommandEnv: command,
			}), nil, writer, writer, s.log.ErrorStreamOnly())
		} else {
			err = runCommand(ctx, "command", s.config.Exec.Command, ToEnvironment(s.workspace, map[string]string{
				provider.CommandEnv: command,
			}), nil, writer, writer, s.log.ErrorStreamOnly())
		}
		if err != nil {
			return err
		}
		s.log.Infof("Successfully stopped container...")

		return nil
	}

	return NewServerClient(s.config, s.server, s.log).Stop(ctx, options)
}

func (s *agentClient) Command(ctx context.Context, commandOptions client.CommandOptions) (err error) {
	if s.workspace.Server.ID == "" {
		// resolve options
		s.m.Lock()
		s.workspace, err = options.ResolveAndSaveOptions(ctx, "command", "", s.workspace, s.config)
		if err != nil {
			s.m.Unlock()
			return err
		}
		environ := ToEnvironment(s.workspace, map[string]string{
			provider.CommandEnv: commandOptions.Command,
		})
		s.m.Unlock()

		// resolve after again
		defer func() {
			s.m.Lock()
			defer s.m.Unlock()

			if err == nil {
				s.workspace, err = options.ResolveAndSaveOptions(ctx, "", "command", s.workspace, s.config)
			}
		}()

		return runCommand(ctx, "command", s.config.Exec.Command, environ, commandOptions.Stdin, commandOptions.Stdout, commandOptions.Stderr, s.log.ErrorStreamOnly())
	}

	// resolve options
	s.m.Lock()
	s.server, err = options.ResolveAndSaveOptionsServer(ctx, "command", "", s.server, s.config)
	if err != nil {
		s.m.Unlock()
		return err
	}
	environ := ToEnvironmentServer(s.server, map[string]string{
		provider.CommandEnv: commandOptions.Command,
	})
	s.m.Unlock()

	// resolve after again
	defer func() {
		if err == nil {
			s.m.Lock()
			defer s.m.Unlock()

			s.server, err = options.ResolveAndSaveOptionsServer(ctx, "", "command", s.server, s.config)
		}
	}()

	return runCommand(ctx, "command", s.config.Exec.Command, environ, commandOptions.Stdin, commandOptions.Stdout, commandOptions.Stderr, s.log.ErrorStreamOnly())
}

func (s *agentClient) Status(ctx context.Context, options client.StatusOptions) (client.Status, error) {
	s.m.Lock()
	defer s.m.Unlock()

	// check if provider has status command
	if len(s.config.Exec.Create) > 0 {
		if s.server == nil {
			return client.StatusNotFound, nil
		}

		return NewServerClient(s.config, s.server, s.log).Status(ctx, options)
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

func ToEnvironment(workspace *provider.Workspace, extraEnv map[string]string) []string {
	env := provider.ToOptions(workspace)

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
