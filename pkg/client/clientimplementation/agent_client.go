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
	"github.com/loft-sh/devpod/pkg/machine"
	"github.com/loft-sh/devpod/pkg/options"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/shell"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"sync"
)

func NewAgentClient(devPodConfig *config.Config, prov *provider.ProviderConfig, workspace *provider.Workspace, log log.Logger) (client.AgentClient, error) {
	var machineConfig *provider.Machine
	if workspace.Machine.ID != "" {
		var err error
		machineConfig, err = provider.LoadMachineConfig(workspace.Context, workspace.Machine.ID)
		if err != nil {
			return nil, errors.Wrap(err, "load machine config")
		}
	}

	return &agentClient{
		devPodConfig: devPodConfig,
		config:       prov,
		workspace:    workspace,
		machine:      machineConfig,
		log:          log,
	}, nil
}

type agentClient struct {
	m sync.Mutex

	devPodConfig *config.Config
	config       *provider.ProviderConfig
	workspace    *provider.Workspace
	machine      *provider.Machine
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

func (s *agentClient) Machine() string {
	if s.machine != nil {
		return s.machine.ID
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
		Machine:   s.machine,
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

	// provider doesn't support machines
	if !s.isMachineProvider() {
		return nil
	}

	// create a new machine
	if s.workspace.Machine.ID != "" {
		return nil
	}

	// create a new machine
	s.workspace = provider.CloneWorkspace(s.workspace)
	s.workspace.Machine.ID = s.workspace.ID
	s.workspace.Machine.AutoDelete = true

	// create machine folder
	var err error
	s.machine, err = machine.CreateMachine(s.workspace.Context, s.workspace.ID, s.config.Name)
	if err != nil {
		return err
	}

	// save workspace config
	err = provider.SaveWorkspaceConfig(s.workspace)
	if err != nil {
		return err
	}

	// create machine
	machineClient, err := NewMachineClient(s.devPodConfig, s.config, s.machine, s.log)
	if err != nil {
		return err
	}

	return machineClient.Create(ctx, options)
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
	if !s.isMachineProvider() || !s.workspace.Machine.AutoDelete {
		writer := s.log.Writer(logrus.InfoLevel, false)
		defer writer.Close()

		s.log.Infof("Deleting container...")
		command := fmt.Sprintf("%s agent workspace delete --id %s --context %s", options.ResolveAgentConfig(s.devPodConfig, s.config).Path, s.workspace.ID, s.workspace.Context)
		err := runCommand(ctx, "command", s.config.Exec.Command, ToEnvironment(s.workspace, s.machine, s.devPodConfig.ProviderOptions(s.config.Name), map[string]string{
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
	} else if s.workspace.Machine.ID != "" && len(s.config.Exec.Delete) > 0 {
		// delete machine if config was found
		machineClient, err := NewMachineClient(s.devPodConfig, s.config, s.machine, s.log)
		if err != nil {
			return err
		}

		err = machineClient.Delete(ctx, opt)
		if err != nil {
			return err
		}
	}

	return DeleteWorkspaceFolder(s.workspace.Context, s.workspace.ID)
}

func (s *agentClient) Start(ctx context.Context, options client.StartOptions) error {
	s.m.Lock()
	defer s.m.Unlock()

	if !s.isMachineProvider() {
		return nil
	}

	machineClient, err := NewMachineClient(s.devPodConfig, s.config, s.machine, s.log)
	if err != nil {
		return err
	}

	return machineClient.Start(ctx, options)
}

func (s *agentClient) Stop(ctx context.Context, opt client.StopOptions) error {
	s.m.Lock()
	defer s.m.Unlock()

	if !s.isMachineProvider() || !s.workspace.Machine.AutoDelete {
		writer := s.log.Writer(logrus.InfoLevel, false)
		defer writer.Close()

		// TODO: stop whole machine if there is no other workspace container running anymore

		s.log.Infof("Stopping container...")
		command := fmt.Sprintf("%s agent workspace stop --id %s --context %s", options.ResolveAgentConfig(s.devPodConfig, s.config).Path, s.workspace.ID, s.workspace.Context)
		err := runCommand(ctx, "command", s.config.Exec.Command, ToEnvironment(s.workspace, s.machine, s.devPodConfig.ProviderOptions(s.config.Name), map[string]string{
			provider.CommandEnv: command,
		}), nil, writer, writer, s.log.ErrorStreamOnly())
		if err != nil {
			return err
		}
		s.log.Infof("Successfully stopped container...")

		return nil
	}

	machineClient, err := NewMachineClient(s.devPodConfig, s.config, s.machine, s.log)
	if err != nil {
		return err
	}

	return machineClient.Stop(ctx, opt)
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
	environ := ToEnvironment(s.workspace, s.machine, s.devPodConfig.ProviderOptions(s.config.Name), map[string]string{
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
	if s.isMachineProvider() {
		if s.machine == nil {
			return client.StatusNotFound, nil
		}

		machineClient, err := NewMachineClient(s.devPodConfig, s.config, s.machine, s.log)
		if err != nil {
			return client.StatusNotFound, err
		}

		status, err := machineClient.Status(ctx, options)
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
	err := runCommand(ctx, "command", s.config.Exec.Command, ToEnvironment(s.workspace, s.machine, s.devPodConfig.ProviderOptions(s.config.Name), map[string]string{
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

func (s *agentClient) isMachineProvider() bool {
	return len(s.config.Exec.Create) > 0
}

func ToEnvironment(workspace *provider.Workspace, machine *provider.Machine, options map[string]config.OptionValue, extraEnv map[string]string) []string {
	env := provider.ToOptions(workspace, machine, options)

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

func DeleteMachineFolder(context, machineID string) error {
	machineDir, err := provider.GetMachineDir(context, machineID)
	if err != nil {
		return err
	}

	// remove machine folder
	err = os.RemoveAll(machineDir)
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
