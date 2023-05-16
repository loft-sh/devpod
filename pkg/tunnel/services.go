package tunnel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/docker/go-connections/nat"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	config2 "github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/netstat"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

func RunInContainer(
	ctx context.Context,
	workspaceClient client.WorkspaceClient,
	devPodConfig *config.Config,
	hostClient,
	containerClient *ssh.Client,
	user string,
	forwardPorts,
	gitCredentials,
	dockerCredentials bool,
	extraPorts []string,
	log log.Logger,
) error {
	// forward ports
	forwardedPorts, err := forwardDevContainerPorts(ctx, workspaceClient, hostClient, containerClient, extraPorts, log)
	if err != nil {
		return errors.Wrap(err, "forward ports")
	}

	dockerCredentials = dockerCredentials && devPodConfig.ContextOption(config.ContextOptionInjectDockerCredentials) == "true"
	gitCredentials = gitCredentials && devPodConfig.ContextOption(config.ContextOptionInjectGitCredentials) == "true"
	forwardPorts = forwardPorts && devPodConfig.ContextOption(config.ContextOptionAutoPortForwarding) == "true"

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	defer stdoutWriter.Close()

	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	defer stdinWriter.Close()

	// start server on stdio
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// run credentials server
	errChan := make(chan error, 1)
	go func() {
		defer cancel()
		writer := log.ErrorStreamOnly().Writer(logrus.DebugLevel, false)
		defer writer.Close()

		command := fmt.Sprintf("%s agent container credentials-server --user '%s'", agent.ContainerDevPodHelperLocation, user)
		if gitCredentials {
			command += " --configure-git-helper"
		}
		if dockerCredentials {
			command += " --configure-docker-helper"
		}
		if forwardPorts {
			command += " --forward-ports"
		}
		if log.GetLevel() == logrus.DebugLevel {
			command += " --debug"
		}

		errChan <- devssh.Run(cancelCtx, containerClient, command, stdinReader, stdoutWriter, writer)
	}()

	// create a port forwarder
	var forwarder netstat.Forwarder
	if forwardPorts {
		forwarder = newForwarder(containerClient, forwardedPorts, log)
	}

	// forward credentials to container
	_, err = agent.RunTunnelServer(
		cancelCtx,
		stdoutReader,
		stdinWriter,
		false,
		gitCredentials,
		dockerCredentials,
		nil,
		forwarder,
		log,
	)
	if err != nil {
		return errors.Wrap(err, "run tunnel server")
	}

	// wait until command finished
	return <-errChan
}

func forwardDevContainerPorts(ctx context.Context, workspaceClient client.WorkspaceClient, sshClient, containerClient *ssh.Client, extraPorts []string, log log.Logger) ([]string, error) {
	result, err := getDevContainerResult(ctx, workspaceClient, sshClient, log)
	if err != nil {
		return nil, fmt.Errorf("error retrieving dev container result: %w", err)
	}

	// extra ports first
	appPorts := []string{}
	appPorts = append(appPorts, result.MergedConfig.AppPort...)
	appPorts = append(appPorts, extraPorts...)

	// return forwarded ports
	forwardedPorts := []string{}

	// app ports
	for _, port := range appPorts {
		parsed, err := nat.ParsePortSpec(port)
		if err != nil {
			log.Debugf("Error parsing appPort %s: %v", port, err)
			continue
		}

		// try to forward
		for _, parsedPort := range parsed {
			if parsedPort.Binding.HostIP == "" {
				parsedPort.Binding.HostIP = "localhost"
			}
			if parsedPort.Binding.HostPort == "" {
				parsedPort.Binding.HostPort = parsedPort.Port.Port()
			}
			go func(parsedPort nat.PortMapping) {
				// do the forward
				log.Debugf("Forward port %s:%s", parsedPort.Binding.HostIP+":"+parsedPort.Binding.HostPort, "localhost:"+parsedPort.Port.Port())
				err = devssh.PortForward(ctx, containerClient, parsedPort.Binding.HostIP+":"+parsedPort.Binding.HostPort, "localhost:"+parsedPort.Port.Port(), log)
				if err != nil {
					log.Debugf("Error port forwarding %s:%s:%s: %v", parsedPort.Binding.HostIP, parsedPort.Binding.HostPort, parsedPort.Port.Port(), err)
				}
			}(parsedPort)

			forwardedPorts = append(forwardedPorts, parsedPort.Binding.HostPort)
		}
	}

	// forward ports
	for _, port := range result.MergedConfig.ForwardPorts {
		// convert port
		i, err := port.Int64()
		if err != nil {
			log.Debugf("Error parsing forwardPort %s: %v", port.String(), err)
			continue
		}

		// try to forward
		go func(i int64, port json.Number) {
			log.Debugf("Forward port %s", port.String())
			err = devssh.PortForward(ctx, containerClient, "localhost:"+port.String(), "localhost:"+port.String(), log)
			if err != nil {
				log.Debugf("Error port forwarding %d: %v", int(i), err)
			}
		}(i, port)

		forwardedPorts = append(forwardedPorts, port.String())
	}

	return forwardedPorts, nil
}

func getDevContainerResult(ctx context.Context, workspaceClient client.WorkspaceClient, client *ssh.Client, log log.Logger) (*config2.Result, error) {
	writer := log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	agentConfig := workspaceClient.AgentConfig()

	// retrieve devcontainer result
	command := fmt.Sprintf("%s agent workspace get-result --id '%s' --context '%s'", workspaceClient.AgentPath(), workspaceClient.Workspace(), workspaceClient.Context())
	if log.GetLevel() == logrus.DebugLevel {
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
	result := &config2.Result{}
	err = json.Unmarshal(buf.Bytes(), result)
	if err != nil {
		return nil, err
	} else if result.MergedConfig == nil {
		return nil, fmt.Errorf("received empty devcontainer result: %v", buf.String())
	}

	return result, nil
}
