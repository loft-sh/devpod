package devcontainer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/agent/tunnelserver"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/driver"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	gosshagent "golang.org/x/crypto/ssh/agent"
)

func (r *runner) setupContainer(
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

	// ssh tunnel
	sshCmd := fmt.Sprintf("'%s' helper ssh-server --stdio", agent.ContainerDevPodHelperLocation)
	if r.Log.GetLevel() == logrus.DebugLevel {
		sshCmd += " --debug"
	}

	// setup container
	r.Log.Infof("Setup container...")
	setupCommand := fmt.Sprintf("'%s' agent container setup --setup-info '%s' --container-workspace-info '%s'", agent.ContainerDevPodHelperLocation, compressed, workspaceConfigCompressed)
	if runtime.GOOS == "linux" || !isDockerDriver {
		setupCommand += " --chown-workspace"
	}
	if !isDockerDriver {
		setupCommand += " --stream-mounts"
	}
	if r.Log.GetLevel() == logrus.DebugLevel {
		setupCommand += " --debug"
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

	errChan := make(chan error, 2)
	go func() {
		defer r.Log.Debugf("Done executing ssh server helper command")
		defer cancel()

		writer := r.Log.Writer(logrus.InfoLevel, false)
		defer writer.Close()

		r.Log.Debugf("Run command in container: %s", sshCmd)
		err = r.Driver.CommandDevContainer(cancelCtx, r.ID, "root", sshCmd, stdinReader, stdoutWriter, writer)
		if err != nil && !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "signal: ") {
			errChan <- fmt.Errorf("executing container command: %w", err)
		} else {
			errChan <- nil
		}
	}()

	// create pipes
	stdoutReader2, stdoutWriter2, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	stdinReader2, stdinWriter2, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer stdoutWriter2.Close()
	defer stdinWriter2.Close()

	go func() {
		defer cancel()

		r.Log.Debugf("Attempting to create SSH client")
		// start ssh client as root / default user
		sshClient, err := devssh.StdioClient(stdoutReader, stdinWriter, false)
		if err != nil {
			errChan <- errors.Wrap(err, "create ssh client")
			return
		}
		defer r.Log.Debugf("Connection to SSH Server closed")
		defer sshClient.Close()

		r.Log.Debugf("SSH client created")

		sess, err := sshClient.NewSession()
		if err != nil {
			errChan <- errors.Wrap(err, "create ssh session")
		}
		defer sess.Close()

		r.Log.Debugf("SSH session created")

		var identityAgent string
		if identityAgent == "" {
			identityAgent = os.Getenv("SSH_AUTH_SOCK")
		}

		if identityAgent != "" {
			err = gosshagent.ForwardToRemote(sshClient, identityAgent)
			if err != nil {
				errChan <- errors.Wrap(err, "forward agent")
			}
			err = gosshagent.RequestAgentForwarding(sess)
			if err != nil {
				errChan <- errors.Wrap(err, "request agent forwarding failed")
			}
		}

		writer := r.Log.Writer(logrus.InfoLevel, false)
		defer writer.Close()

		err = devssh.Run(ctx, sshClient, setupCommand, stdinReader2, stdoutWriter2, writer)
		if err != nil {
			errChan <- errors.Wrap(err, "run agent command")
		} else {
			errChan <- nil
		}
	}()

	// start server
	result, err = tunnelserver.RunSetupServer(
		cancelCtx,
		stdoutReader2,
		stdinWriter2,
		r.WorkspaceConfig.Agent.InjectDockerCredentials != "false",
		config.GetMounts(result),
		r.Log,
	)
	if err != nil {
		return nil, errors.Wrap(err, "run tunnel machine")
	}

	// wait until command finished
	if err := <-errChan; err != nil {
		return result, err
	}

	return result, <-errChan
}
