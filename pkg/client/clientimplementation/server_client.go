package clientimplementation

import (
	"bytes"
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/provider/options"
	"github.com/loft-sh/devpod/pkg/types"
	"io"
	"os"
	"strings"
)

func NewServerClient(devPodConfig *config.Config, provider *provider.ProviderConfig, server *provider.Server, log log.Logger) client.Client {
	return &serverClient{
		devPodConfig: devPodConfig,
		config:       provider,
		server:       server,
		log:          log,
	}
}

type serverClient struct {
	devPodConfig *config.Config
	config       *provider.ProviderConfig
	server       *provider.Server
	log          log.Logger
}

func (s *serverClient) Provider() string {
	return s.config.Name
}

func (s *serverClient) ProviderType() provider.ProviderType {
	return s.config.Type
}

func (s *serverClient) Context() string {
	return s.server.Context
}

func (s *serverClient) Create(ctx context.Context, options client.CreateOptions) error {
	// create a server
	s.log.Infof("Create %s server...", s.config.Name)
	err := runCommand(ctx, "create", s.config.Exec.Create, ToEnvironment(nil, s.server, s.devPodConfig.ProviderOptions(s.config.Name), nil), os.Stdin, os.Stdout, os.Stderr, s.log)
	if err != nil {
		return err
	}

	s.log.Donef("Successfully created %s server", s.config.Name)
	return nil
}

func (s *serverClient) Start(ctx context.Context, options client.StartOptions) error {
	err := runCommand(ctx, "start", s.config.Exec.Start, ToEnvironment(nil, s.server, s.devPodConfig.ProviderOptions(s.config.Name), nil), os.Stdin, os.Stdout, os.Stderr, s.log)
	if err != nil {
		return err
	}

	return nil
}

func (s *serverClient) Stop(ctx context.Context, options client.StopOptions) error {
	err := runCommand(ctx, "stop", s.config.Exec.Stop, ToEnvironment(nil, s.server, s.devPodConfig.ProviderOptions(s.config.Name), nil), os.Stdin, os.Stdout, os.Stderr, s.log)
	if err != nil {
		return err
	}

	return nil
}

func (s *serverClient) Command(ctx context.Context, commandOptions client.CommandOptions) error {
	var err error

	// resolve options
	_, s.devPodConfig, err = options.ResolveAndSaveOptions(ctx, "command", "", nil, s.server, s.devPodConfig, s.config)
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			_, s.devPodConfig, err = options.ResolveAndSaveOptions(ctx, "", "command", nil, s.server, s.devPodConfig, s.config)
		}
	}()

	return runCommand(ctx, "command", s.config.Exec.Command, ToEnvironment(nil, s.server, s.devPodConfig.ProviderOptions(s.config.Name), map[string]string{
		provider.CommandEnv: commandOptions.Command,
	}), commandOptions.Stdin, commandOptions.Stdout, commandOptions.Stderr, s.log.ErrorStreamOnly())
}

func (s *serverClient) Status(ctx context.Context, options client.StatusOptions) (client.Status, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := runCommand(ctx, "status", s.config.Exec.Status, ToEnvironment(nil, s.server, s.devPodConfig.ProviderOptions(s.config.Name), nil), nil, stdout, stderr, s.log)
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

func (s *serverClient) Delete(ctx context.Context, options client.DeleteOptions) error {
	// kill the command after the grace period
	if options.GracePeriod != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *options.GracePeriod)
		defer cancel()
	}

	s.log.Infof("Deleting %s server...", s.config.Name)
	err := runCommand(ctx, "delete", s.config.Exec.Delete, ToEnvironment(nil, s.server, s.devPodConfig.ProviderOptions(s.config.Name), nil), os.Stdin, os.Stdout, os.Stderr, s.log)
	if err != nil {
		if !options.Force {
			return err
		}

		s.log.Errorf("Error deleting server %s", s.server.ID)
	}
	s.log.Donef("Successfully deleted %s server", s.config.Name)

	// delete server folder
	err = DeleteServerFolder(s.server.Context, s.server.ID)
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
