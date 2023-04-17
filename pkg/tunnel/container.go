package tunnel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/log"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/token"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

func NewContainerTunnel(client client.WorkspaceClient, log log.Logger) *ContainerHandler {
	updateConfigInterval := time.Second * 30
	workspacePortForwarding := true
	return &ContainerHandler{
		client:                  client,
		updateConfigInterval:    updateConfigInterval,
		workspacePortForwarding: workspacePortForwarding,
		log:                     log,
	}
}

type ContainerHandler struct {
	client                  client.WorkspaceClient
	updateConfigInterval    time.Duration
	workspacePortForwarding bool
	log                     log.Logger
}

type Handler func(ctx context.Context, client *ssh.Client) error

func (c *ContainerHandler) Run(ctx context.Context, runInHost Handler, runInContainer Handler) error {
	if runInHost == nil && runInContainer == nil {
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

	// get token
	tok, err := token.GetDevPodToken()
	if err != nil {
		return err
	}

	// tunnel to host
	tunnelChan := make(chan error, 1)
	go func() {
		writer := c.log.ErrorStreamOnly().Writer(logrus.InfoLevel, false)
		defer writer.Close()
		defer c.log.Debugf("Tunnel to host closed")

		command := fmt.Sprintf("%s helper ssh-server --token '%s' --stdio", c.client.AgentPath(), tok)
		if c.log.GetLevel() == logrus.DebugLevel {
			command += " --debug"
		}
		tunnelChan <- agent.InjectAgentAndExecute(cancelCtx, func(ctx context.Context, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			return c.client.Command(ctx, client.CommandOptions{
				Command: command,
				Stdin:   stdin,
				Stdout:  stdout,
				Stderr:  stderr,
			})
		}, c.client.AgentPath(), c.client.AgentURL(), true, command, stdinReader, stdoutWriter, writer, c.log.ErrorStreamOnly())
	}()

	privateKey, err := devssh.GetDevPodPrivateKeyRaw()
	if err != nil {
		return err
	}

	// connect to container
	containerChan := make(chan error, 2)
	go func() {
		// start ssh client as root / default user
		sshClient, err := devssh.StdioClientFromKeyBytes(privateKey, stdoutReader, stdinWriter, false)
		if err != nil {
			containerChan <- errors.Wrap(err, "create ssh client")
			return
		}
		defer sshClient.Close()
		defer cancel()
		defer c.log.Debugf("Connection to container closed")
		c.log.Debugf("Successfully connected to host")

		// do port-forwarding etc. here with sshClient
		waitGroup := sync.WaitGroup{}
		if runInContainer != nil {
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				defer c.log.Debugf("Run in container done")

				containerChan <- errors.Wrap(c.runRunInContainer(cancelCtx, sshClient, tok, privateKey, runInContainer), "run in container")
			}()
		}

		// tunnel to host
		if runInHost != nil {
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				defer c.log.Debugf("Run in host done")

				containerChan <- errors.Wrap(runInHost(cancelCtx, sshClient), "run in host")
			}()
		}

		// update workspace remotely
		if c.updateConfigInterval > 0 {
			go func() {
				c.updateConfig(cancelCtx, sshClient)
			}()
		}

		// wait until we are done
		waitGroup.Wait()
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
			workspaceInfo, agentInfo, err := c.client.AgentInfo()
			if err != nil {
				c.log.Errorf("Error compressing workspace info: %v", err)
				break
			}

			// update workspace remotely
			buf := &bytes.Buffer{}
			command := fmt.Sprintf("%s agent workspace update-config --workspace-info '%s'", c.client.AgentPath(), workspaceInfo)
			if agentInfo.Agent.DataPath != "" {
				command += fmt.Sprintf(" --agent-dir '%s'", agentInfo.Agent.DataPath)
			}

			c.log.Debugf("Run command in container: %s", command)
			err = devssh.Run(ctx, sshClient, command, nil, buf, buf)
			if err != nil {
				c.log.Errorf("Error updating remote workspace: %s%v", buf.String(), err)
			} else {
				c.log.Debugf("Out: %s", buf.String())
			}
		}
	}
}

func (c *ContainerHandler) runRunInContainer(ctx context.Context, sshClient *ssh.Client, tok string, privateKey []byte, runInContainer Handler) error {
	// compress info
	workspaceInfo, _, err := c.client.AgentInfo()
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

		command := fmt.Sprintf("%s agent container-tunnel --token '%s' --workspace-info '%s'", c.client.AgentPath(), tok, workspaceInfo)
		if c.log.GetLevel() == logrus.DebugLevel {
			command += " --debug"
		}
		err = devssh.Run(cancelCtx, sshClient, command, stdinReader, stdoutWriter, writer)
		if err != nil {
			c.log.Errorf("Error tunneling to container: %v", err)
			return
		}
	}()

	// start ssh client
	containerClient, err := devssh.StdioClientFromKeyBytes(privateKey, stdoutReader, stdinWriter, false)
	if err != nil {
		return err
	}
	defer containerClient.Close()
	c.log.Debugf("Successfully connected to container")

	// run port-forwarding
	if c.workspacePortForwarding {
		go func() {
			// start forwarding ports
			c.forwardPorts(cancelCtx, sshClient, containerClient)
		}()
	}

	// start handler
	return runInContainer(cancelCtx, containerClient)
}

func (c *ContainerHandler) forwardPorts(ctx context.Context, sshClient, containerClient *ssh.Client) {
	result, err := c.getDevContainerResult(ctx, sshClient)
	if err != nil {
		c.log.Errorf("Error retrieving dev container result: %v", err)
		return
	}

	// app ports
	for _, port := range result.MergedConfig.AppPort {
		parsed, err := nat.ParsePortSpec(port)
		if err != nil {
			c.log.Debugf("Error parsing appPort %s: %v", port, err)
			continue
		}

		// try to forward
		for _, parsedPort := range parsed {
			go func(parsedPort nat.PortMapping) {
				if parsedPort.Binding.HostIP == "" {
					parsedPort.Binding.HostIP = "localhost"
				}
				if parsedPort.Binding.HostPort == "" {
					parsedPort.Binding.HostPort = parsedPort.Port.Port()
				}

				// do the forward
				err = devssh.PortForward(ctx, containerClient, parsedPort.Binding.HostIP+":"+parsedPort.Binding.HostPort, "localhost:"+parsedPort.Port.Port(), c.log)
				if err != nil {
					c.log.Debugf("Error port forwarding %s:%s:%s: %v", parsedPort.Binding.HostIP, parsedPort.Binding.HostPort, parsedPort.Port.Port(), err)
				}
			}(parsedPort)
		}
	}

	// forward ports
	for _, port := range result.MergedConfig.ForwardPorts {
		// convert port
		i, err := port.Int64()
		if err != nil {
			c.log.Debugf("Error parsing forwardPort %s: %v", port.String(), err)
			continue
		}

		// try to forward
		go func(i int64, port json.Number) {
			err = devssh.PortForward(ctx, containerClient, "localhost:"+port.String(), "localhost:"+port.String(), c.log)
			if err != nil {
				c.log.Debugf("Error port forwarding %d: %v", int(i), err)
			}
		}(i, port)
	}
}

func (c *ContainerHandler) getDevContainerResult(ctx context.Context, client *ssh.Client) (*config.Result, error) {
	writer := c.log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	agentConfig := c.client.AgentConfig()

	// retrieve devcontainer result
	command := fmt.Sprintf("%s agent workspace get-result --id '%s' --context '%s'", c.client.AgentPath(), c.client.Workspace(), c.client.Context())
	if c.log.GetLevel() == logrus.DebugLevel {
		command += " --debug"
	}
	if agentConfig.DataPath != "" {
		command += fmt.Sprintf(" --agent-dir '%s'", agentConfig.DataPath)
	}
	buf := &bytes.Buffer{}
	err := devssh.Run(ctx, client, command, nil, buf, writer)
	if err != nil {
		return nil, fmt.Errorf("error retrieving workspace get-result: %w", err)
	}

	// parse result
	result := &config.Result{}
	err = json.Unmarshal(buf.Bytes(), result)
	if err != nil {
		return nil, err
	} else if result.MergedConfig == nil {
		return nil, fmt.Errorf("received empty devcontainer result: %v", buf.String())
	}

	return result, nil
}
