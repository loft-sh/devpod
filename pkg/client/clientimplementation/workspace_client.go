package clientimplementation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/pkg/binaries"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
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

func NewWorkspaceClient(devPodConfig *config.Config, prov *provider.ProviderConfig, workspace *provider.Workspace, machine *provider.Machine, log log.Logger) (client.WorkspaceClient, error) {
	if workspace.Machine.ID != "" && machine == nil {
		return nil, fmt.Errorf("workspace machine is not found")
	} else if prov.IsMachineProvider() && workspace.Machine.ID == "" {
		return nil, fmt.Errorf("workspace machine ID is empty, but machine provider found")
	}

	return &workspaceClient{
		devPodConfig: devPodConfig,
		config:       prov,
		workspace:    workspace,
		machine:      machine,
		log:          log,
	}, nil
}

type workspaceClient struct {
	m sync.Mutex

	devPodConfig *config.Config
	config       *provider.ProviderConfig
	workspace    *provider.Workspace
	machine      *provider.Machine
	log          log.Logger
}

func (s *workspaceClient) Provider() string {
	return s.config.Name
}

func (s *workspaceClient) Workspace() string {
	s.m.Lock()
	defer s.m.Unlock()

	return s.workspace.ID
}

func (s *workspaceClient) WorkspaceConfig() *provider.Workspace {
	s.m.Lock()
	defer s.m.Unlock()

	return provider.CloneWorkspace(s.workspace)
}

func (s *workspaceClient) AgentPath() string {
	s.m.Lock()
	defer s.m.Unlock()

	return options.ResolveAgentConfig(s.devPodConfig, s.config, s.workspace, s.machine).Path
}

func (s *workspaceClient) AgentURL() string {
	s.m.Lock()
	defer s.m.Unlock()

	return options.ResolveAgentConfig(s.devPodConfig, s.config, s.workspace, s.machine).DownloadURL
}

func (s *workspaceClient) Context() string {
	return s.workspace.Context
}

func (s *workspaceClient) RefreshOptions(ctx context.Context, userOptionsRaw []string) error {
	s.m.Lock()
	defer s.m.Unlock()

	userOptions, err := provider.ParseOptions(s.config, userOptionsRaw)
	if err != nil {
		return errors.Wrap(err, "parse options")
	}

	if s.isMachineProvider() {
		if s.machine == nil {
			return nil
		}

		machine, err := options.ResolveAndSaveOptionsMachine(ctx, s.devPodConfig, s.config, s.machine, userOptions, s.log)
		if err != nil {
			return err
		}

		s.machine = machine
		return nil
	}

	workspace, err := options.ResolveAndSaveOptionsWorkspace(ctx, s.devPodConfig, s.config, s.workspace, userOptions, s.log)
	if err != nil {
		return err
	}

	s.workspace = workspace
	return nil
}

func (s *workspaceClient) AgentConfig() provider.ProviderAgentConfig {
	s.m.Lock()
	defer s.m.Unlock()

	return options.ResolveAgentConfig(s.devPodConfig, s.config, s.workspace, s.machine)
}

func (s *workspaceClient) AgentInfo() (string, *provider.AgentWorkspaceInfo, error) {
	s.m.Lock()
	defer s.m.Unlock()

	agentInfo := &provider.AgentWorkspaceInfo{
		Workspace: s.workspace,
		Machine:   s.machine,
		Agent:     options.ResolveAgentConfig(s.devPodConfig, s.config, s.workspace, s.machine),
		Options:   s.devPodConfig.ProviderOptions(s.Provider()),
	}

	// marshal config
	out, err := json.Marshal(agentInfo)
	if err != nil {
		return "", nil, err
	}

	compressed, err := compress.Compress(string(out))
	if err != nil {
		return "", nil, err
	}

	return compressed, agentInfo, nil
}

func (s *workspaceClient) Create(ctx context.Context, options client.CreateOptions) error {
	s.m.Lock()
	defer s.m.Unlock()

	// provider doesn't support machines
	if !s.isMachineProvider() {
		return nil
	}

	// check machine state
	if s.machine == nil {
		return fmt.Errorf("machine is not defined")
	}

	// create machine client
	machineClient, err := NewMachineClient(s.devPodConfig, s.config, s.machine, s.log)
	if err != nil {
		return err
	}

	// get status
	machineStatus, err := machineClient.Status(ctx, client.StatusOptions{})
	if err != nil {
		return err
	} else if machineStatus != client.StatusNotFound {
		return nil
	}

	// create the machine
	return machineClient.Create(ctx, client.CreateOptions{})
}

func (s *workspaceClient) Delete(ctx context.Context, opt client.DeleteOptions) error {
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
		agentConfig := options.ResolveAgentConfig(s.devPodConfig, s.config, s.workspace, s.machine)
		command := fmt.Sprintf("%s agent workspace delete --id %s --context %s", agentConfig.Path, s.workspace.ID, s.workspace.Context)
		if agentConfig.DataPath != "" {
			command += fmt.Sprintf(" --agent-dir '%s'", agentConfig.DataPath)
		}
		err := RunCommandWithBinaries(
			ctx,
			"command",
			s.config.Exec.Command,
			s.workspace.Context,
			s.workspace,
			s.machine,
			s.devPodConfig.ProviderOptions(s.config.Name),
			s.config,
			map[string]string{
				provider.CommandEnv: command,
			},
			nil,
			writer,
			writer,
			s.log.ErrorStreamOnly(),
		)
		if err != nil {
			if !opt.Force {
				return err
			}

			if err != context.DeadlineExceeded {
				s.log.Errorf("Error deleting container: %v", err)
			}
		}
	} else if s.machine != nil && s.workspace.Machine.ID != "" && len(s.config.Exec.Delete) > 0 {
		// delete machine if config was found
		machineClient, err := NewMachineClient(s.devPodConfig, s.config, s.machine, s.log)
		if err != nil {
			if !opt.Force {
				return err
			}
		}

		err = machineClient.Delete(ctx, opt)
		if err != nil {
			return err
		}
	}

	return DeleteWorkspaceFolder(s.workspace.Context, s.workspace.ID)
}

func (s *workspaceClient) Start(ctx context.Context, options client.StartOptions) error {
	s.m.Lock()
	defer s.m.Unlock()

	if !s.isMachineProvider() || s.machine == nil {
		return nil
	}

	machineClient, err := NewMachineClient(s.devPodConfig, s.config, s.machine, s.log)
	if err != nil {
		return err
	}

	return machineClient.Start(ctx, options)
}

func (s *workspaceClient) Stop(ctx context.Context, opt client.StopOptions) error {
	s.m.Lock()
	defer s.m.Unlock()

	if !s.isMachineProvider() || !s.workspace.Machine.AutoDelete {
		writer := s.log.Writer(logrus.InfoLevel, false)
		defer writer.Close()

		s.log.Infof("Stopping container...")
		agentConfig := options.ResolveAgentConfig(s.devPodConfig, s.config, s.workspace, s.machine)
		command := fmt.Sprintf("%s agent workspace stop --id '%s' --context '%s'", agentConfig.Path, s.workspace.ID, s.workspace.Context)
		if agentConfig.DataPath != "" {
			command += fmt.Sprintf(" --agent-dir '%s'", agentConfig.DataPath)
		}
		err := RunCommandWithBinaries(
			ctx,
			"command",
			s.config.Exec.Command,
			s.workspace.Context,
			s.workspace,
			s.machine,
			s.devPodConfig.ProviderOptions(s.config.Name),
			s.config,
			map[string]string{
				provider.CommandEnv: command,
			},
			nil,
			writer,
			writer,
			s.log.ErrorStreamOnly(),
		)
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

func (s *workspaceClient) Command(ctx context.Context, commandOptions client.CommandOptions) (err error) {
	// get environment variables
	s.m.Lock()
	environ, err := binaries.ToEnvironmentWithBinaries(s.workspace.Context, s.workspace, s.machine, s.devPodConfig.ProviderOptions(s.config.Name), s.config, map[string]string{
		provider.CommandEnv: commandOptions.Command,
	}, s.log)
	if err != nil {
		return err
	}
	s.m.Unlock()

	// resolve options
	return runCommand(ctx, "command", s.config.Exec.Command, environ, commandOptions.Stdin, commandOptions.Stdout, commandOptions.Stderr, s.log.ErrorStreamOnly())
}

func (s *workspaceClient) Status(ctx context.Context, options client.StatusOptions) (client.Status, error) {
	s.m.Lock()
	defer s.m.Unlock()

	// check if provider has status command
	if s.isMachineProvider() && len(s.config.Exec.Status) > 0 {
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

func (s *workspaceClient) getContainerStatus(ctx context.Context) (client.Status, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	agentConfig := options.ResolveAgentConfig(s.devPodConfig, s.config, s.workspace, s.machine)
	command := fmt.Sprintf("%s agent workspace status --id '%s' --context '%s'", agentConfig.Path, s.workspace.ID, s.workspace.Context)
	if agentConfig.DataPath != "" {
		command += fmt.Sprintf(" --agent-dir '%s'", agentConfig.DataPath)
	}
	err := RunCommandWithBinaries(ctx, "command", s.config.Exec.Command, s.workspace.Context, s.workspace, s.machine, s.devPodConfig.ProviderOptions(s.config.Name), s.config, map[string]string{
		provider.CommandEnv: command,
	}, nil, stdout, stderr, s.log.ErrorStreamOnly())
	if err != nil {
		return client.StatusNotFound, err
	}

	parsed, err := client.ParseStatus(stdout.String())
	if err != nil {
		return client.StatusNotFound, fmt.Errorf("error parsing container status: %s%s%v", stdout.String(), stderr.String(), err)
	}

	return parsed, nil
}

func (s *workspaceClient) isMachineProvider() bool {
	return len(s.config.Exec.Create) > 0
}

func RunCommandWithBinaries(ctx context.Context, name string, command types.StrArray, context string, workspace *provider.Workspace, machine *provider.Machine, options map[string]config.OptionValue, config *provider.ProviderConfig, extraEnv map[string]string, stdin io.Reader, stdout io.Writer, stderr io.Writer, log log.Logger) (err error) {
	environ, err := binaries.ToEnvironmentWithBinaries(context, workspace, machine, options, config, extraEnv, log)
	if err != nil {
		return err
	}

	return runCommand(ctx, name, command, environ, stdin, stdout, stderr, log)
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
