package clientimplementation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofrs/flock"
	"github.com/loft-sh/devpod/pkg/binaries"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/config"
	config2 "github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/options"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/shell"
	"github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/loft-sh/log"
	perrors "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

	workspaceLockOnce sync.Once
	workspaceLock     *flock.Flock
	machineLock       *flock.Flock

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

func (s *workspaceClient) AgentLocal() bool {
	s.m.Lock()
	defer s.m.Unlock()

	return options.ResolveAgentConfig(s.devPodConfig, s.config, s.workspace, s.machine).Local == "true"
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

func (s *workspaceClient) RefreshOptions(ctx context.Context, userOptionsRaw []string, reconfigure bool) error {
	s.m.Lock()
	defer s.m.Unlock()

	userOptions, err := provider.ParseOptions(userOptionsRaw)
	if err != nil {
		return perrors.Wrap(err, "parse options")
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

func (s *workspaceClient) AgentInjectGitCredentials(cliOptions provider.CLIOptions) bool {
	s.m.Lock()
	defer s.m.Unlock()

	return s.agentInfo(cliOptions).Agent.InjectGitCredentials == "true"
}

func (s *workspaceClient) AgentInjectDockerCredentials(cliOptions provider.CLIOptions) bool {
	s.m.Lock()
	defer s.m.Unlock()

	return s.agentInfo(cliOptions).Agent.InjectDockerCredentials == "true"
}

func (s *workspaceClient) AgentInfo(cliOptions provider.CLIOptions) (string, *provider.AgentWorkspaceInfo, error) {
	s.m.Lock()
	defer s.m.Unlock()

	return s.compressedAgentInfo(cliOptions)
}

func (s *workspaceClient) compressedAgentInfo(cliOptions provider.CLIOptions) (string, *provider.AgentWorkspaceInfo, error) {
	agentInfo := s.agentInfo(cliOptions)

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

func (s *workspaceClient) agentInfo(cliOptions provider.CLIOptions) *provider.AgentWorkspaceInfo {
	// try to load last devcontainer.json
	var lastDevContainerConfig *config2.DevContainerConfigWithPath
	var workspaceOrigin string
	if s.workspace != nil {
		result, err := provider.LoadWorkspaceResult(s.workspace.Context, s.workspace.ID)
		if err != nil {
			s.log.Debugf("Error loading workspace result: %v", err)
		} else if result != nil {
			lastDevContainerConfig = result.DevContainerConfigWithPath
		}

		workspaceOrigin = s.workspace.Origin
	}

	// build struct
	agentInfo := &provider.AgentWorkspaceInfo{
		WorkspaceOrigin:        workspaceOrigin,
		Workspace:              s.workspace,
		Machine:                s.machine,
		LastDevContainerConfig: lastDevContainerConfig,
		CLIOptions:             cliOptions,
		Agent:                  options.ResolveAgentConfig(s.devPodConfig, s.config, s.workspace, s.machine),
		Options:                s.devPodConfig.ProviderOptions(s.Provider()),
	}

	// if we are running platform mode
	if cliOptions.Platform.Enabled {
		agentInfo.Agent.InjectGitCredentials = "true"
		agentInfo.Agent.InjectDockerCredentials = "true"
	}

	// we don't send any provider options if proxy because these could contain
	// sensitive information and we don't want to allow privileged containers that
	// have access to the host to save these.
	if agentInfo.Agent.Driver != provider.CustomDriver && (cliOptions.Platform.Enabled || cliOptions.DisableDaemon) {
		agentInfo.Options = map[string]config.OptionValue{}
		agentInfo.Workspace = provider.CloneWorkspace(agentInfo.Workspace)
		agentInfo.Workspace.Provider.Options = map[string]config.OptionValue{}
		if agentInfo.Machine != nil {
			agentInfo.Machine = provider.CloneMachine(agentInfo.Machine)
			agentInfo.Machine.Provider.Options = map[string]config.OptionValue{}
		}
	}

	// Get the timeout from the context options
	agentInfo.InjectTimeout = config.ParseTimeOption(s.devPodConfig, config.ContextOptionAgentInjectTimeout)

	// Set registry cache from context option
	agentInfo.RegistryCache = s.devPodConfig.ContextOption(config.ContextOptionRegistryCache)

	return agentInfo
}

func (s *workspaceClient) initLock() {
	s.workspaceLockOnce.Do(func() {
		s.m.Lock()
		defer s.m.Unlock()

		// get locks dir
		workspaceLocksDir, err := provider.GetLocksDir(s.workspace.Context)
		if err != nil {
			panic(fmt.Errorf("get workspaces dir: %w", err))
		}
		_ = os.MkdirAll(workspaceLocksDir, 0777)

		// create workspace lock
		s.workspaceLock = flock.New(filepath.Join(workspaceLocksDir, s.workspace.ID+".workspace.lock"))

		// create machine lock
		if s.machine != nil {
			s.machineLock = flock.New(filepath.Join(workspaceLocksDir, s.machine.ID+".machine.lock"))
		}
	})
}

func (s *workspaceClient) Lock(ctx context.Context) error {
	s.initLock()

	// try to lock workspace
	s.log.Debugf("Acquire workspace lock...")
	err := tryLock(ctx, s.workspaceLock, "workspace", s.log)
	if err != nil {
		return fmt.Errorf("error locking workspace: %w", err)
	}
	s.log.Debugf("Acquired workspace lock...")

	// try to lock machine
	if s.machineLock != nil {
		s.log.Debugf("Acquire machine lock...")
		err := tryLock(ctx, s.machineLock, "machine", s.log)
		if err != nil {
			return fmt.Errorf("error locking machine: %w", err)
		}
		s.log.Debugf("Acquired machine lock...")
	}

	return nil
}

func (s *workspaceClient) Unlock() {
	s.initLock()

	// try to unlock machine
	if s.machineLock != nil {
		err := s.machineLock.Unlock()
		if err != nil {
			s.log.Warnf("Error unlocking machine: %v", err)
		}
	}

	// try to unlock workspace
	err := s.workspaceLock.Unlock()
	if err != nil {
		s.log.Warnf("Error unlocking workspace: %v", err)
	}
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

	// parse duration
	var gracePeriod *time.Duration
	if opt.GracePeriod != "" {
		duration, err := time.ParseDuration(opt.GracePeriod)
		if err == nil {
			gracePeriod = &duration
		}
	}

	// kill the command after the grace period
	if gracePeriod != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *gracePeriod)
		defer cancel()
	}

	// should just delete container?
	if !s.isMachineProvider() || !s.workspace.Machine.AutoDelete {
		isRunning, err := s.isMachineRunning(ctx)
		if err != nil {
			if !opt.Force {
				return err
			}
		} else if isRunning {
			writer := s.log.Writer(logrus.InfoLevel, false)
			defer writer.Close()

			s.log.Infof("Deleting container...")
			compressed, info, err := s.compressedAgentInfo(provider.CLIOptions{})
			if err != nil {
				return fmt.Errorf("agent info")
			}
			command := fmt.Sprintf("'%s' agent workspace delete --workspace-info '%s'", info.Agent.Path, compressed)
			err = RunCommandWithBinaries(
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

				if !errors.Is(err, context.DeadlineExceeded) {
					s.log.Errorf("Error deleting container: %v", err)
				}
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

	return DeleteWorkspaceFolder(s.workspace.Context, s.workspace.ID, s.workspace.SSHConfigPath, s.log)
}

func (s *workspaceClient) isMachineRunning(ctx context.Context) (bool, error) {
	if !s.isMachineProvider() {
		return true, nil
	}

	// delete machine if config was found
	machineClient, err := NewMachineClient(s.devPodConfig, s.config, s.machine, s.log)
	if err != nil {
		return false, err
	}

	// retrieve status
	status, err := machineClient.Status(ctx, client.StatusOptions{})
	if err != nil {
		return false, perrors.Wrap(err, "retrieve machine status")
	} else if status == client.StatusRunning {
		return true, nil
	}

	return false, nil
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
		compressed, info, err := s.compressedAgentInfo(provider.CLIOptions{})
		if err != nil {
			return fmt.Errorf("agent info")
		}
		command := fmt.Sprintf("'%s' agent workspace stop --workspace-info '%s'", info.Agent.Path, compressed)
		err = RunCommandWithBinaries(
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
	buf := &bytes.Buffer{}
	compressed, info, err := s.compressedAgentInfo(provider.CLIOptions{})
	if err != nil {
		return "", fmt.Errorf("get agent info")
	}
	command := fmt.Sprintf("'%s' agent workspace status --workspace-info '%s'", info.Agent.Path, compressed)
	err = RunCommandWithBinaries(ctx, "command", s.config.Exec.Command, s.workspace.Context, s.workspace, s.machine, s.devPodConfig.ProviderOptions(s.config.Name), s.config, map[string]string{
		provider.CommandEnv: command,
	}, nil, io.MultiWriter(stdout, buf), buf, s.log.ErrorStreamOnly())
	if err != nil {
		return client.StatusNotFound, fmt.Errorf("error retrieving container status: %s%w", buf.String(), err)
	}

	parsed, err := client.ParseStatus(stdout.String())
	if err != nil {
		return client.StatusNotFound, fmt.Errorf("error parsing container status: %s%w", buf.String(), err)
	}

	s.log.Debugf("Container status command output (stdout & stderr): %s %s (%s)", buf.String(), stdout.String(), parsed)
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
		return shell.RunEmulatedShell(ctx, command[0], stdin, stdout, stderr, environ)
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

func DeleteWorkspaceFolder(context string, workspaceID string, sshConfigPath string, log log.Logger) error {
	path, err := ssh.ResolveSSHConfigPath(sshConfigPath)
	if err != nil {
		return err
	}
	sshConfigPath = path

	err = ssh.RemoveFromConfig(workspaceID, sshConfigPath, log)
	if err != nil {
		log.Errorf("Remove workspace '%s' from ssh config: %v", workspaceID, err)
	}

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
