package sshtunnel

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/loft-sh/log"

	"github.com/loft-sh/devpod/pkg/agent/tunnelserver"
	client2 "github.com/loft-sh/devpod/pkg/client"
	config2 "github.com/loft-sh/devpod/pkg/devcontainer/config"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	devsshagent "github.com/loft-sh/devpod/pkg/ssh/agent"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type AgentInjectFunc func(context.Context, string, *os.File, *os.File, io.WriteCloser) error

// ExecuteCommand runs the command in an SSH Tunnel and returns the result.
func ExecuteCommand(
	ctx context.Context,
	client client2.WorkspaceClient,
	agentInject AgentInjectFunc,
	sshCommand,
	command string,
	proxy,
	setupContainer,
	allowGitCredentials,
	allowDockerCredentials bool,
	mounts []*config2.Mount,
	gitUsername string,
	gitToken string,
	log log.Logger,
) (*config2.Result, error) {
	// create pipes
	sshTunnelStdoutReader, sshTunnelStdoutWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	sshTunnelStdinReader, sshTunnelStdinWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer sshTunnelStdoutWriter.Close()
	defer sshTunnelStdinWriter.Close()

	// start machine on stdio
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error, 2)
	go func() {
		defer log.Debugf("Done executing ssh server helper command")
		defer cancel()

		writer := log.Writer(logrus.InfoLevel, false)
		defer writer.Close()

		log.Debugf("Inject and run command: %s", sshCommand)
		err := agentInject(ctx, sshCommand, sshTunnelStdinReader, sshTunnelStdoutWriter, writer)
		if err != nil && !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "signal: ") {
			errChan <- fmt.Errorf("executing agent command: %w", err)
		} else {
			errChan <- nil
		}
	}()

	// create pipes
	gRPCConnStdoutReader, gRPCConnStdoutWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	gRPCConnStdinReader, gRPCConnStdinWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer gRPCConnStdoutWriter.Close()
	defer gRPCConnStdinWriter.Close()

	// connect to container
	go func() {
		defer cancel()

		log.Debugf("Attempting to create SSH client")
		// start ssh client as root / default user
		sshClient, err := devssh.StdioClient(sshTunnelStdoutReader, sshTunnelStdinWriter, false)
		if err != nil {
			errChan <- errors.Wrap(err, "create ssh client")
			return
		}
		defer log.Debugf("Connection to SSH Server closed")
		defer sshClient.Close()

		log.Debugf("SSH client created")

		sess, err := sshClient.NewSession()
		if err != nil {
			errChan <- errors.Wrap(err, "create ssh session")
		}
		defer sess.Close()

		log.Debugf("SSH session created")

		identityAgent := devsshagent.GetSSHAuthSocket()
		if identityAgent != "" {
			err = devsshagent.ForwardToRemote(sshClient, identityAgent)
			if err != nil {
				errChan <- errors.Wrap(err, "forward agent")
			}
			err = devsshagent.RequestAgentForwarding(sess)
			if err != nil {
				errChan <- errors.Wrap(err, "request agent forwarding failed")
			}
		}

		writer := log.Writer(logrus.InfoLevel, false)
		defer writer.Close()

		err = devssh.Run(ctx, sshClient, command, gRPCConnStdinReader, gRPCConnStdoutWriter, writer)
		if err != nil {
			errChan <- errors.Wrap(err, "run agent command")
		} else {
			errChan <- nil
		}
	}()

	// create container etc.
	var result *config2.Result
	if proxy {
		// create client on stdin & stdout
		tunnelClient, err := tunnelserver.NewTunnelClient(os.Stdin, os.Stdout, true)
		if err != nil {
			return nil, errors.Wrap(err, "create tunnel client")
		}

		// create proxy server
		result, err = tunnelserver.RunProxyServer(
			cancelCtx,
			tunnelClient,
			gRPCConnStdoutReader,
			gRPCConnStdinWriter,
			log,
			gitUsername,
			gitToken,
		)
		if err != nil {
			return nil, errors.Wrap(err, "run proxy tunnel")
		}
	} else if setupContainer {
		// start server
		result, err = tunnelserver.RunSetupServer(
			cancelCtx,
			gRPCConnStdoutReader,
			gRPCConnStdinWriter,
			allowGitCredentials,
			allowDockerCredentials,
			mounts,
			log,
		)
		if err != nil {
			return nil, errors.Wrap(err, "run tunnel machine")
		}
	} else {
		result, err = tunnelserver.RunUpServer(
			cancelCtx,
			gRPCConnStdoutReader,
			gRPCConnStdinWriter,
			client.AgentInjectGitCredentials(),
			client.AgentInjectDockerCredentials(),
			client.WorkspaceConfig(),
			log,
			tunnelserver.WithGitCredentialsOverride(gitUsername, gitToken),
		)
		if err != nil {
			return nil, errors.Wrap(err, "run tunnel machine")
		}
	}

	// wait until command finished
	if err := <-errChan; err != nil {
		return result, err
	}

	return result, <-errChan
}
