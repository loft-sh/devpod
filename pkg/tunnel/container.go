package tunnel

import (
	"bytes"
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/log"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/token"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"sync"
	"time"
)

func NewContainerTunnel(client client.AgentClient, log log.Logger) *ContainerHandler {
	updateConfigInterval := time.Second * 30
	return &ContainerHandler{
		client:               client,
		updateConfigInterval: updateConfigInterval,
		log:                  log,
	}
}

type ContainerHandler struct {
	client               client.AgentClient
	updateConfigInterval time.Duration
	log                  log.Logger
}

type Handler func(client *ssh.Client) error

func (c *ContainerHandler) Run(ctx context.Context, runInHost Handler, runInContainer Handler) error {
	if runInHost == nil && runInContainer == nil {
		return nil
	}

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
	tok, err := token.GenerateTemporaryToken()
	if err != nil {
		return err
	}

	// tunnel to host
	//TODO: right now we have a tunnel in a tunnel, maybe its better to start 2 separate commands?
	tunnelChan := make(chan error, 1)
	go func() {
		writer := c.log.ErrorStreamOnly().Writer(logrus.InfoLevel, false)
		defer writer.Close()

		command := fmt.Sprintf("%s helper ssh-server --token '%s' --stdio", c.client.AgentPath(), tok)
		if c.log.GetLevel() == logrus.DebugLevel {
			command += " --debug"
		}
		tunnelChan <- agent.InjectAgentAndExecute(ctx, func(ctx context.Context, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			return c.client.Command(ctx, client.CommandOptions{
				Command: command,
				Stdin:   stdin,
				Stdout:  stdout,
				Stderr:  stderr,
			})
		}, c.client.AgentPath(), c.client.AgentURL(), true, command, stdinReader, stdoutWriter, writer, c.log.ErrorStreamOnly())
	}()

	privateKey, err := devssh.GetTempPrivateKeyRaw()
	if err != nil {
		return err
	}

	// connect to container
	containerChan := make(chan error, 2)
	go func() {
		// start ssh client as root / default user
		sshClient, err := devssh.StdioClientFromKeyBytes(privateKey, stdoutReader, stdinWriter, false)
		if err != nil {
			containerChan <- err
			return
		}
		defer sshClient.Close()
		c.log.Debugf("Successfully connected to host")

		// do port-forwarding etc. here with sshClient
		waitGroup := sync.WaitGroup{}
		if runInContainer != nil {
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()

				containerChan <- c.runRunInContainer(sshClient, tok, privateKey, runInContainer)
			}()
		}

		// tunnel to host
		if runInHost != nil {
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()

				containerChan <- runInHost(sshClient)
			}()
		}

		// update workspace remotely
		doneChan := make(chan struct{})
		if c.updateConfigInterval > 0 {
			go func() {
				c.updateConfig(ctx, sshClient, doneChan)
			}()
		}

		// wait until we are done
		waitGroup.Wait()
		close(doneChan)
	}()

	// wait for result
	select {
	case err := <-containerChan:
		return errors.Wrap(err, "tunnel to container")
	case err := <-tunnelChan:
		return errors.Wrap(err, "connect to server")
	}
}

func (c *ContainerHandler) updateConfig(ctx context.Context, sshClient *ssh.Client, doneChan chan struct{}) {
	for {
		select {
		case <-doneChan:
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
			workspaceInfo, err := c.client.AgentInfo()
			if err != nil {
				c.log.Errorf("Error compressing workspace info: %v", err)
				break
			}

			// update workspace remotely
			buf := &bytes.Buffer{}
			command := fmt.Sprintf("%s agent workspace update-config --workspace-info '%s'", c.client.AgentPath(), workspaceInfo)
			c.log.Debugf("Run command in container: %s", command)
			err = devssh.Run(sshClient, command, nil, buf, buf)
			if err != nil {
				c.log.Errorf("Error updating remote workspace: %s%v", buf.String(), err)
			} else {
				c.log.Debugf("Out: %s", buf.String())
			}
		}
	}
}

func (c *ContainerHandler) runRunInContainer(sshClient *ssh.Client, tok string, privateKey []byte, runInContainer Handler) error {
	// compress info
	workspaceInfo, err := c.client.AgentInfo()
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

	// tunnel to container
	go func() {
		writer := c.log.Writer(logrus.InfoLevel, false)
		defer writer.Close()

		c.log.Debugf("Run container tunnel")
		defer c.log.Debugf("Container tunnel exited")

		command := fmt.Sprintf("%s agent container-tunnel --token '%s' --workspace-info '%s'", c.client.AgentPath(), tok, workspaceInfo)
		if c.log.GetLevel() == logrus.DebugLevel {
			command += " --debug"
		}
		err = devssh.Run(sshClient, command, stdinReader, stdoutWriter, writer)
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

	// start handler
	return runInContainer(containerClient)
}
