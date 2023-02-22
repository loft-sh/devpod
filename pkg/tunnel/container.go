package tunnel

import (
	"bytes"
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/provider/options"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/token"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"os"
	"sync"
	"time"
)

func NewContainerTunnel(provider provider2.ServerProvider, workspace *provider2.Workspace, log log.Logger) *ContainerHandler {
	updateConfigInterval := time.Second * 5
	return &ContainerHandler{
		workspace:            workspace,
		provider:             provider,
		updateConfigInterval: updateConfigInterval,
		log:                  log,
	}
}

type ContainerHandler struct {
	workspaceMutex sync.Mutex
	workspace      *provider2.Workspace

	provider             provider2.ServerProvider
	updateConfigInterval time.Duration
	log                  log.Logger
}

type Handler func(client *ssh.Client) error

func (c *ContainerHandler) Run(ctx context.Context, runInHost Handler, runInContainer Handler) error {
	if runInHost == nil && runInContainer == nil {
		return nil
	}

	// get token
	tok, err := token.GenerateTemporaryToken()
	if err != nil {
		return err
	}
	privateKey, err := devssh.GetTempPrivateKeyRaw()
	if err != nil {
		return err
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

	// tunnel to host
	//TODO: right now we have a tunnel in a tunnel, maybe its better to start 2 separate commands?
	tunnelChan := make(chan error, 1)
	go func() {
		tunnelChan <- c.provider.Command(ctx, c.getWorkspace(), provider2.CommandOptions{
			Command: fmt.Sprintf("%s helper ssh-server --token '%s' --stdio", c.getWorkspace().Provider.Agent.Path, tok),
			Stdin:   stdinReader,
			Stdout:  stdoutWriter,
			Stderr:  os.Stderr,
		})
	}()

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
				c.updateConfig(sshClient, doneChan)
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

func (c *ContainerHandler) getWorkspace() *provider2.Workspace {
	c.workspaceMutex.Lock()
	defer c.workspaceMutex.Unlock()

	return c.workspace
}

func (c *ContainerHandler) setWorkspace(w *provider2.Workspace) {
	c.workspaceMutex.Lock()
	defer c.workspaceMutex.Unlock()

	c.workspace = w
}

func (c *ContainerHandler) updateConfig(sshClient *ssh.Client, doneChan chan struct{}) {
	for {
		select {
		case <-doneChan:
			return
		case <-time.After(c.updateConfigInterval):
			// update options
			workspace, err := options.ResolveAndSaveOptions(context.Background(), "command", "", c.getWorkspace(), c.provider)
			if err != nil {
				c.log.Errorf("Error resolving options: %v", err)
				break
			}

			// update workspace
			c.setWorkspace(workspace)

			// compress info
			workspaceInfo, err := provider2.NewAgentWorkspaceInfo(c.getWorkspace())
			if err != nil {
				c.log.Errorf("Error compressing workspace info: %v", err)
				break
			}

			// update workspace remotely
			buf := &bytes.Buffer{}
			err = devssh.Run(sshClient, fmt.Sprintf("%s agent update-config --workspace-info '%s'", c.getWorkspace().Provider.Agent.Path, workspaceInfo), nil, buf, buf)
			if err != nil {
				c.log.Errorf("Error updating remote workspace: %s%v", buf.String(), err)
			}
		}
	}
}

func (c *ContainerHandler) runRunInContainer(sshClient *ssh.Client, tok string, privateKey []byte, runInContainer Handler) error {
	// compress info
	workspaceInfo, err := provider2.NewAgentWorkspaceInfo(c.getWorkspace())
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
	containerChan := make(chan error, 1)
	go func() {
		buf := &bytes.Buffer{}
		err = devssh.Run(sshClient, fmt.Sprintf("%s agent container-tunnel --token '%s' --workspace-info '%s'", c.getWorkspace().Provider.Agent.Path, tok, workspaceInfo), stdinReader, stdoutWriter, os.Stderr)
		if err != nil {
			c.log.Errorf("Error tunneling to container: %v", err)
			containerChan <- errors.Wrapf(err, "%s", buf.String())
			return
		}
		c.log.Debugf("Container tunnel exited")
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
