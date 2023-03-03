package clientimplementation

import (
	"bytes"
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/options"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/types"
	"io"
	"os"
	"strings"
)

func NewMachineClient(devPodConfig *config.Config, provider *provider.ProviderConfig, machine *provider.Machine, log log.Logger) (client.Client, error) {
	if !provider.IsMachineProvider() {
		return nil, fmt.Errorf("provider '%s' is not a machine provider. Please use another provider", provider.Name)
	}

	return &machineClient{
		devPodConfig: devPodConfig,
		config:       provider,
		machine:      machine,
		log:          log,
	}, nil
}

type machineClient struct {
	devPodConfig *config.Config
	config       *provider.ProviderConfig
	machine      *provider.Machine
	log          log.Logger
}

func (s *machineClient) Provider() string {
	return s.config.Name
}

func (s *machineClient) ProviderType() provider.ProviderType {
	return s.config.Type
}

func (s *machineClient) Machine() string {
	return s.machine.ID
}

func (s *machineClient) AgentPath() string {
	return options.ResolveAgentConfig(s.devPodConfig, s.config).Path
}

func (s *machineClient) AgentURL() string {
	return options.ResolveAgentConfig(s.devPodConfig, s.config).DownloadURL
}

func (s *machineClient) Context() string {
	return s.machine.Context
}

func (s *machineClient) Create(ctx context.Context, options client.CreateOptions) error {
	// create a machine
	s.log.Infof("Create machine '%s' with provider '%s'...", s.machine.ID, s.config.Name)
	err := runCommand(ctx, "create", s.config.Exec.Create, ToEnvironment(nil, s.machine, s.devPodConfig.ProviderOptions(s.config.Name), nil), os.Stdin, os.Stdout, os.Stderr, s.log)
	if err != nil {
		return err
	}

	s.log.Donef("Successfully created machine '%s' with provider '%s'", s.machine.ID, s.config.Name)
	return nil
}

func (s *machineClient) Start(ctx context.Context, options client.StartOptions) error {
	s.log.Infof("Starting machine '%s'...", s.machine.ID)
	err := runCommand(ctx, "start", s.config.Exec.Start, ToEnvironment(nil, s.machine, s.devPodConfig.ProviderOptions(s.config.Name), nil), os.Stdin, os.Stdout, os.Stderr, s.log)
	if err != nil {
		return err
	}
	s.log.Donef("Successfully started '%s'", s.machine.ID)

	return nil
}

func (s *machineClient) Stop(ctx context.Context, options client.StopOptions) error {
	s.log.Infof("Stopping machine '%s'...", s.machine.ID)
	err := runCommand(ctx, "stop", s.config.Exec.Stop, ToEnvironment(nil, s.machine, s.devPodConfig.ProviderOptions(s.config.Name), nil), os.Stdin, os.Stdout, os.Stderr, s.log)
	if err != nil {
		return err
	}
	s.log.Donef("Successfully stopped '%s'", s.machine.ID)

	return nil
}

func (s *machineClient) Command(ctx context.Context, commandOptions client.CommandOptions) error {
	var err error

	// resolve options
	s.devPodConfig, err = options.ResolveAndSaveOptions(ctx, "command", "", s.devPodConfig, s.config)
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			s.devPodConfig, err = options.ResolveAndSaveOptions(ctx, "", "command", s.devPodConfig, s.config)
		}
	}()

	return runCommand(ctx, "command", s.config.Exec.Command, ToEnvironment(nil, s.machine, s.devPodConfig.ProviderOptions(s.config.Name), map[string]string{
		provider.CommandEnv: commandOptions.Command,
	}), commandOptions.Stdin, commandOptions.Stdout, commandOptions.Stderr, s.log.ErrorStreamOnly())
}

func (s *machineClient) Status(ctx context.Context, options client.StatusOptions) (client.Status, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := runCommand(ctx, "status", s.config.Exec.Status, ToEnvironment(nil, s.machine, s.devPodConfig.ProviderOptions(s.config.Name), nil), nil, stdout, stderr, s.log)
	if err != nil {
		return client.StatusNotFound, fmt.Errorf("get status: %s%s", strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()))
	}

	// parse status
	parsedStatus, err := client.ParseStatus(stdout.String())
	if err != nil {
		return client.StatusNotFound, err
	}

	return parsedStatus, nil
}

func (s *machineClient) Delete(ctx context.Context, options client.DeleteOptions) error {
	// kill the command after the grace period
	if options.GracePeriod != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *options.GracePeriod)
		defer cancel()
	}

	s.log.Infof("Deleting %s machine...", s.config.Name)
	err := runCommand(ctx, "delete", s.config.Exec.Delete, ToEnvironment(nil, s.machine, s.devPodConfig.ProviderOptions(s.config.Name), nil), os.Stdin, os.Stdout, os.Stderr, s.log)
	if err != nil {
		if !options.Force {
			return err
		}

		s.log.Errorf("Error deleting machine %s", s.machine.ID)
	}
	s.log.Donef("Successfully deleted %s machine", s.config.Name)

	// delete machine folder
	err = DeleteMachineFolder(s.machine.Context, s.machine.ID)
	if err != nil {
		return err
	}

	return nil
}

func runCommand(ctx context.Context, name string, command types.StrArray, environ []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, log log.Logger) (err error) {
	if len(command) == 0 {
		return nil
	}

	// log
	log.Debugf("Run %s provider command: %s", name, strings.Join(command, " "))

	// run the command
	return RunCommand(ctx, command, environ, stdin, stdout, stderr)
}
