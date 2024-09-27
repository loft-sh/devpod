package tunnel

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/provider"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

func NewContainerTunnel(client client.WorkspaceClient, proxy bool, log log.Logger) *ContainerHandler {
	updateConfigInterval := time.Second * 30
	return &ContainerHandler{
		client:               client,
		updateConfigInterval: updateConfigInterval,
		proxy:                proxy,
		log:                  log,
	}
}

type ContainerHandler struct {
	client               client.WorkspaceClient
	updateConfigInterval time.Duration
	proxy                bool
	log                  log.Logger
}

type Handler func(ctx context.Context, containerClient *ssh.Client) error

func (c *ContainerHandler) Run(ctx context.Context, handler Handler, cfg *config.Config, envVars map[string]string) error {
	if handler == nil {
		return nil
	}

	// create context
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// create readers
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	defer stdoutWriter.Close()
	defer stdinWriter.Close()

	// Get the timeout from the context options
	timeout := config.ParseTimeOption(cfg, config.ContextOptionAgentInjectTimeout)

	// tunnel to host
	tunnelChan := make(chan error, 1)
	go func() {
		writer := c.log.ErrorStreamOnly().Writer(logrus.InfoLevel, false)
		defer writer.Close()
		defer c.log.Debugf("Tunnel to host closed")

		command := fmt.Sprintf("'%s' helper ssh-server --stdio", c.client.AgentPath())
		if c.log.GetLevel() == logrus.DebugLevel {
			command += " --debug"
		}
		tunnelChan <- agent.InjectAgentAndExecute(
			cancelCtx,
			func(ctx context.Context, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
				return c.client.Command(ctx, client.CommandOptions{
					Command: command,
					Stdin:   stdin,
					Stdout:  stdout,
					Stderr:  stderr,
				})
			},
			c.client.AgentLocal(),
			c.client.AgentPath(),
			c.client.AgentURL(),
			true,
			command,
			stdinReader,
			stdoutWriter,
			writer,
			c.log.ErrorStreamOnly(),
			timeout)
	}()

	// connect to container
	containerChan := make(chan error, 1)
	go func() {
		// start ssh client as root / default user
		sshClient, err := devssh.StdioClient(stdoutReader, stdinWriter, false)
		if err != nil {
			containerChan <- errors.Wrap(err, "create ssh client")
			return
		}

		defer sshClient.Close()
		defer cancel()
		defer c.log.Debugf("Connection to container closed")
		c.log.Debugf("Successfully connected to host")

		// update workspace remotely
		if !c.proxy && c.updateConfigInterval > 0 {
			go func() {
				c.updateConfig(cancelCtx, sshClient)
			}()
		}

		// wait until we are done
		if err := c.runRunInContainer(cancelCtx, sshClient, handler, envVars); err != nil {
			containerChan <- fmt.Errorf("run in container: %w", err)
		} else {
			containerChan <- nil
		}
	}()

	// wait for result
	select {
	case err := <-containerChan:
		return errors.Wrap(err, "tunnel to container")
	case err := <-tunnelChan:
		return errors.Wrap(err, "connect to server")
	}
}

func (c *ContainerHandler) updateConfig(ctx context.Context, sshClient *ssh.Client) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(c.updateConfigInterval):
			c.log.Debugf("Start refresh")

			// update options
			err := c.client.RefreshOptions(ctx, nil)
			if err != nil {
				c.log.Errorf("Error refreshing workspace options: %v", err)
				break
			}

			// compress info
			workspaceInfo, agentInfo, err := c.client.AgentInfo(provider.CLIOptions{Proxy: c.proxy})
			if err != nil {
				c.log.Errorf("Error compressing workspace info: %v", err)
				break
			}

			// update workspace remotely
			buf := &bytes.Buffer{}
			command := fmt.Sprintf("'%s' agent workspace update-config --workspace-info '%s'", c.client.AgentPath(), workspaceInfo)
			if agentInfo.Agent.DataPath != "" {
				command += fmt.Sprintf(" --agent-dir '%s'", agentInfo.Agent.DataPath)
			}

			c.log.Debugf("Run command in container: %s", command)
			err = devssh.Run(ctx, sshClient, command, nil, buf, buf, nil)
			if err != nil {
				c.log.Errorf("Error updating remote workspace: %s%v", buf.String(), err)
			} else {
				c.log.Debugf("Out: %s", buf.String())
			}
		}
	}
}

func (c *ContainerHandler) runRunInContainer(ctx context.Context, sshClient *ssh.Client, runInContainer Handler, envVars map[string]string) error {
	// compress info
	workspaceInfo, _, err := c.client.AgentInfo(provider.CLIOptions{Proxy: c.proxy})
	if err != nil {
		return err
	}

	// create pipes
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	defer stdoutWriter.Close()
	defer stdinWriter.Close()

	// create cancel context
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// tunnel to container
	go func() {
		writer := c.log.Writer(logrus.InfoLevel, false)
		defer writer.Close()
		defer stdoutWriter.Close()
		defer cancel()

		c.log.Debugf("Run container tunnel")
		defer c.log.Debugf("Container tunnel exited")

		command := fmt.Sprintf("'%s' agent container-tunnel --workspace-info '%s'", c.client.AgentPath(), workspaceInfo)
		if c.log.GetLevel() == logrus.DebugLevel {
			command += " --debug"
		}
		err = devssh.Run(cancelCtx, sshClient, command, stdinReader, stdoutWriter, writer, envVars)
		if err != nil {
			c.log.Errorf("Error tunneling to container: %v", err)
			return
		}
	}()

	// start ssh client
	containerClient, err := devssh.StdioClient(stdoutReader, stdinWriter, false)
	if err != nil {
		return err
	}
	defer containerClient.Close()
	c.log.Debugf("Successfully connected to container")

	// start handler
	return runInContainer(cancelCtx, containerClient)
}
