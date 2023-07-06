package tunnel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/docker/go-connections/nat"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/agent/tunnelserver"
	"github.com/loft-sh/devpod/pkg/config"
	config2 "github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/setup"
	"github.com/loft-sh/devpod/pkg/ide/openvscode"
	"github.com/loft-sh/devpod/pkg/netstat"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

func RunInContainer(
	ctx context.Context,
	devPodConfig *config.Config,
	containerClient *ssh.Client,
	user string,
	forwardPorts bool,
	gitCredentials,
	dockerCredentials bool,
	extraPorts []string,
	log log.Logger,
) error {
	// forward ports
	forwardedPorts, err := forwardDevContainerPorts(ctx, containerClient, extraPorts, log)
	if err != nil {
		return errors.Wrap(err, "forward ports")
	}

	dockerCredentials = dockerCredentials && devPodConfig.ContextOption(config.ContextOptionSSHInjectDockerCredentials) == "true"
	gitCredentials = gitCredentials && devPodConfig.ContextOption(config.ContextOptionSSHInjectGitCredentials) == "true"

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

		command := fmt.Sprintf("'%s' agent container credentials-server --user '%s'", agent.ContainerDevPodHelperLocation, user)
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
		forwarder = newForwarder(containerClient, append(forwardedPorts, fmt.Sprintf("%d", openvscode.DefaultVSCodePort)), log)
	}

	// forward credentials to container
	_, err = tunnelserver.RunTunnelServer(
		cancelCtx,
		stdoutReader,
		stdinWriter,
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

func forwardDevContainerPorts(ctx context.Context, containerClient *ssh.Client, extraPorts []string, log log.Logger) ([]string, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := devssh.Run(ctx, containerClient, "cat "+setup.ResultLocation, nil, stdout, stderr)
	if err != nil {
		return nil, fmt.Errorf("retrieve container result: %s\n%s%w", stdout.String(), stderr.String(), err)
	}

	// parse result
	result := &config2.Result{}
	err = json.Unmarshal(stdout.Bytes(), result)
	if err != nil {
		return nil, fmt.Errorf("error parsing container result %s: %w", stdout.String(), err)
	}
	log.Debugf("Successfully parsed result at %s", setup.ResultLocation)

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
		host, portNumber, err := parseForwardPort(port)
		if err != nil {
			log.Debugf("Error parsing forwardPort %s: %v", port, err)
		}

		// try to forward
		go func(port string) {
			log.Debugf("Forward port %s", port)
			err = devssh.PortForward(
				ctx,
				containerClient,
				fmt.Sprintf("localhost:%d", portNumber),
				fmt.Sprintf("%s:%d", host, portNumber),
				log,
			)
			if err != nil {
				log.Debugf("Error port forwarding %s: %v", port, err)
			}
		}(port)

		forwardedPorts = append(forwardedPorts, port)
	}

	return forwardedPorts, nil
}

func parseForwardPort(port string) (string, int64, error) {
	tokens := strings.Split(port, ":")

	if len(tokens) == 1 {
		port, err := strconv.ParseInt(tokens[0], 10, 64)
		if err != nil {
			return "", 0, err
		}
		return "localhost", port, nil
	}

	if len(tokens) == 2 {
		port, err := strconv.ParseInt(tokens[1], 10, 64)
		if err != nil {
			return "", 0, err
		}
		return tokens[0], port, nil
	}

	return "", 0, fmt.Errorf("invalid forwardPorts port")
}
