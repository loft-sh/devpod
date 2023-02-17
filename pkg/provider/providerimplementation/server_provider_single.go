package providerimplementation

import (
	"bytes"
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/config"
	config2 "github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"strings"
)

const (
	singleName = "single"
)

func (s *serverProvider) CreateSingle(ctx context.Context, originalWorkspace *provider.Workspace, options provider.CreateOptions) error {
	if len(s.config.Exec.Create) == 0 {
		return nil
	}

	s.log.Infof("Create %s server...", s.config.Name)
	err := runProviderCommandSingle(ctx, "create", s.config.Exec.Create, originalWorkspace, s, os.Stdin, os.Stdout, os.Stderr, nil, s.log)
	if err != nil {
		return err
	}
	s.log.Donef("Successfully created %s server", s.config.Name)
	return nil
}

func (s *serverProvider) DeleteSingle(ctx context.Context, workspace *provider.Workspace, options provider.DeleteOptions) error {
	agentConfig, err := s.AgentConfig()
	if err != nil {
		return err
	}

	writer := s.log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	s.log.Infof("Deleting container...")
	err = s.CommandSingle(ctx, workspace, provider.CommandOptions{
		Command: fmt.Sprintf("%s agent delete --id %s --context %s", agentConfig.Path, workspace.ID, workspace.Context),
		Stdout:  writer,
		Stderr:  writer,
	})
	if err != nil {
		if !options.Force {
			return err
		}

		s.log.Errorf("Error deleting container: %v", err)
	} else {
		s.log.Infof("Successfully deleted container...")
	}

	return DeleteWorkspaceFolder(workspace.Context, workspace.ID)
}

func (s *serverProvider) StartSingle(ctx context.Context, originalWorkspace *provider.Workspace, options provider.StartOptions) error {
	err := runProviderCommandSingle(ctx, "start", s.config.Exec.Start, originalWorkspace, s, os.Stdin, os.Stdout, os.Stderr, nil, s.log)
	if err != nil {
		return err
	}

	return nil
}

func (s *serverProvider) StopSingle(ctx context.Context, workspace *provider.Workspace, options provider.StopOptions) error {
	agentConfig, err := s.AgentConfig()
	if err != nil {
		return err
	}

	writer := s.log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	// TODO: stop whole machine if there is no other workspace container running anymore

	s.log.Infof("Stopping container...")
	err = s.CommandSingle(ctx, workspace, provider.CommandOptions{
		Command: fmt.Sprintf("%s agent stop --id %s --context %s", agentConfig.Path, workspace.ID, workspace.Context),
		Stdout:  writer,
		Stderr:  writer,
	})
	if err != nil {
		return err
	}
	s.log.Infof("Successfully stopped container...")
	return nil
}

func (s *serverProvider) CommandSingle(ctx context.Context, originalWorkspace *provider.Workspace, options provider.CommandOptions) error {
	err := runProviderCommandSingle(ctx, "command", s.config.Exec.Command, originalWorkspace, s, options.Stdin, options.Stdout, options.Stderr, map[string]string{
		provider.CommandEnv: options.Command,
	}, s.log.ErrorStreamOnly())
	if err != nil {
		return err
	}

	return nil
}

func (s *serverProvider) StatusSingle(ctx context.Context, originalWorkspace *provider.Workspace, options provider.StatusOptions) (provider.Status, error) {
	// check if provider has status command
	if len(s.config.Exec.Status) > 0 {
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		err := runProviderCommandSingle(ctx, "status", s.config.Exec.Status, originalWorkspace, s, nil, stdout, stderr, nil, s.log)
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
	workspaceFolder, err := config.GetWorkspaceDir(originalWorkspace.Context, originalWorkspace.ID)
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

func runProviderCommandSingle(ctx context.Context, name string, command types.StrArray, originalWorkspace *provider.Workspace, prov provider.Provider, stdin io.Reader, stdout io.Writer, stderr io.Writer, extraEnv map[string]string, log log.Logger) (err error) {
	if len(command) == 0 {
		return nil
	}

	// log
	log.Debugf("Run %s provider command: %s", name, strings.Join(command, " "))

	// resolve options
	if originalWorkspace != nil {
		err = resolveOptions(ctx, name, "", originalWorkspace, prov)
		if err != nil {
			return err
		}
		defer func() {
			if err == nil {
				err = resolveOptions(ctx, "", name, originalWorkspace, prov)
			}
		}()
	}

	// transform workspace
	workspace, err := toSingleWorkspace(originalWorkspace)
	if err != nil {
		return err
	}

	// run the command
	return RunCommand(ctx, command, workspace, stdin, stdout, stderr, extraEnv)
}

func toSingleWorkspace(workspace *provider.Workspace) (*provider.Workspace, error) {
	retWorkspace := &provider.Workspace{}
	err := config2.Convert(workspace, retWorkspace)
	if err != nil {
		return nil, err
	}

	// make sure the keys exist
	_, err = ssh.GetTempPrivateKeyRaw()
	if err != nil {
		return nil, err
	}

	retWorkspace.ID = singleName
	retWorkspace.Folder = ssh.GetKeysTempDir()
	return retWorkspace, nil
}
