package devcontainer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/agent/tunnelserver"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/driver"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (r *Runner) setupContainer(
	ctx context.Context,
	containerDetails *config.ContainerDetails,
	mergedConfig *config.MergedDevContainerConfig,
) (*config.Result, error) {
	// inject agent
	err := agent.InjectAgent(ctx, func(ctx context.Context, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
		return r.Driver.CommandDevContainer(ctx, r.ID, "root", command, stdin, stdout, stderr)
	}, false, agent.ContainerDevPodHelperLocation, agent.DefaultAgentDownloadURL(), false, r.Log)
	if err != nil {
		return nil, errors.Wrap(err, "inject agent")
	}
	r.Log.Debugf("Injected into container")
	defer r.Log.Debugf("Done setting up container")

	// compress info
	result := &config.Result{
		MergedConfig:        mergedConfig,
		SubstitutionContext: r.SubstitutionContext,
		ContainerDetails:    containerDetails,
	}
	marshalled, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	compressed, err := compress.Compress(string(marshalled))
	if err != nil {
		return nil, err
	}

	// compress container workspace info
	workspaceConfigRaw, err := json.Marshal(&provider2.ContainerWorkspaceInfo{
		IDE:              r.WorkspaceConfig.Workspace.IDE,
		CLIOptions:       r.WorkspaceConfig.CLIOptions,
		Dockerless:       r.WorkspaceConfig.Agent.Dockerless,
		ContainerTimeout: r.WorkspaceConfig.Agent.ContainerTimeout,
	})
	if err != nil {
		return nil, err
	}
	workspaceConfigCompressed, err := compress.Compress(string(workspaceConfigRaw))
	if err != nil {
		return nil, err
	}

	// check if docker driver
	_, isDockerDriver := r.Driver.(driver.DockerDriver)

	// setup container
	r.Log.Infof("Setup container...")
	command := fmt.Sprintf("'%s' agent container setup --setup-info '%s' --container-workspace-info '%s'", agent.ContainerDevPodHelperLocation, compressed, workspaceConfigCompressed)
	if runtime.GOOS == "linux" || !isDockerDriver {
		command += " --chown-workspace"
	}
	if !isDockerDriver {
		command += " --stream-mounts"
	}
	if r.Log.GetLevel() == logrus.DebugLevel {
		command += " --debug"
	}

	// create pipes
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer stdoutWriter.Close()
	defer stdinWriter.Close()

	// start machine on stdio
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		defer r.Log.Debugf("Done executing up command")
		defer cancel()

		writer := r.Log.Writer(logrus.InfoLevel, false)
		defer writer.Close()

		r.Log.Debugf("Run command in container: %s", command)
		err = r.Driver.CommandDevContainer(cancelCtx, r.ID, "root", command, stdinReader, stdoutWriter, writer)
		if err != nil {
			errChan <- fmt.Errorf("executing container command: %w", err)
		} else {
			errChan <- nil
		}
	}()

	// start server
	result, err = tunnelserver.RunSetupServer(
		cancelCtx,
		stdoutReader,
		stdinWriter,
		r.WorkspaceConfig.Agent.InjectDockerCredentials != "false",
		config.GetMounts(result),
		r.Log,
	)
	if err != nil {
		return nil, errors.Wrap(err, "run tunnel machine")
	}

	return result, <-errChan
}
