package tunnel

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/agent/tunnelserver"
	"github.com/loft-sh/devpod/pkg/config"
	config2 "github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/setup"
	"github.com/loft-sh/devpod/pkg/gitsshsigning"
	"github.com/loft-sh/devpod/pkg/ide/openvscode"
	"github.com/loft-sh/devpod/pkg/netstat"
	"github.com/loft-sh/devpod/pkg/provider"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

// RunServices forwards the ports for a given workspace and uses it's SSH client to run the credentials server remotely and the services server locally to communicate with the container
func RunServices(
	ctx context.Context,
	devPodConfig *config.Config,
	containerClient *ssh.Client,
	user string,
	forwardPorts bool,
	extraPorts []string,
	gitUsername,
	gitToken string,
	workspace *provider.Workspace,
	log log.Logger,
) error {
	// calculate exit after timeout
	exitAfterTimeout := time.Second * 5
	if devPodConfig.ContextOption(config.ContextOptionExitAfterTimeout) != "true" {
		exitAfterTimeout = 0
	}

	// forward ports
	forwardedPorts, err := forwardDevContainerPorts(ctx, containerClient, extraPorts, exitAfterTimeout, log)
	if err != nil {
		return errors.Wrap(err, "forward ports")
	}

	configureDockerCredentials := devPodConfig.ContextOption(config.ContextOptionSSHInjectDockerCredentials) == "true"
	configureGitCredentials := devPodConfig.ContextOption(config.ContextOptionSSHInjectGitCredentials) == "true"
	configureGitSSHSignatureHelper := devPodConfig.ContextOption(config.ContextOptionGitSSHSignatureForwarding) == "true"

	return retry.OnError(wait.Backoff{
		Steps:    math.MaxInt,
		Duration: 500 * time.Millisecond,
		Factor:   1,
		Jitter:   0.1,
	}, func(err error) bool {
		// Always allow to retry. Potentially add exceptions in the future.
		return true
	}, func() error {
		stdoutReader, stdoutWriter, err := os.Pipe()
		if err != nil {
			return err
		}
		defer stdoutWriter.Close()

		stdinReader, stdinWriter, err := os.Pipe()
		if err != nil {
			return err
		}

		// start server on stdio
		cancelCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		// create a port forwarder
		var forwarder netstat.Forwarder
		if forwardPorts {
			forwarder = newForwarder(containerClient, append(forwardedPorts, fmt.Sprintf("%d", openvscode.DefaultVSCodePort)), log)
		}

		errChan := make(chan error, 1)
		go func() {
			defer cancel()
			defer stdinWriter.Close()
			// forward credentials to container
			err := tunnelserver.RunServicesServer(
				cancelCtx,
				stdoutReader,
				stdinWriter,
				configureGitCredentials,
				configureDockerCredentials,
				forwarder,
				workspace,
				log,
				tunnelserver.WithGitCredentialsOverride(gitUsername, gitToken),
			)
			if err != nil {
				errChan <- errors.Wrap(err, "run tunnel server")
			}
			close(errChan)
		}()

		// run credentials server
		writer := log.ErrorStreamOnly().Writer(logrus.DebugLevel, false)
		defer writer.Close()

		command := fmt.Sprintf("'%s' agent container credentials-server --user '%s'", agent.ContainerDevPodHelperLocation, user)
		if configureGitCredentials {
			command += " --configure-git-helper"
		}
		if configureGitSSHSignatureHelper {
			format, userSigningKey, err := gitsshsigning.ExtractGitConfiguration()
			if err == nil && format == gitsshsigning.GPGFormatSSH && userSigningKey != "" {
				encodedKey := base64.StdEncoding.EncodeToString([]byte(userSigningKey))
				command += fmt.Sprintf(" --git-user-signing-key %s", encodedKey)
			}
		}
		if configureDockerCredentials {
			command += " --configure-docker-helper"
		}
		if forwardPorts {
			command += " --forward-ports"
		}
		if log.GetLevel() == logrus.DebugLevel {
			command += " --debug"
		}

		err = devssh.Run(cancelCtx, containerClient, command, stdinReader, stdoutWriter, writer, nil)
		if err != nil {
			return err
		}
		err = <-errChan
		if err != nil {
			return err
		}

		return nil
	})
}

// forwardDevContainerPorts forwards all the ports defined in the devcontainer.json
func forwardDevContainerPorts(ctx context.Context, containerClient *ssh.Client, extraPorts []string, exitAfterTimeout time.Duration, log log.Logger) ([]string, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := devssh.Run(ctx, containerClient, "cat "+setup.ResultLocation, nil, stdout, stderr, nil)
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

	// return forwarded ports
	forwardedPorts := []string{}

	// extra ports
	for _, port := range extraPorts {
		forwardedPorts = append(forwardedPorts, forwardPort(ctx, containerClient, port, exitAfterTimeout, log)...)
	}

	// app ports
	for _, port := range result.MergedConfig.AppPort {
		forwardedPorts = append(forwardedPorts, forwardPort(ctx, containerClient, port, 0, log)...)
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
				"tcp",
				fmt.Sprintf("localhost:%d", portNumber),
				"tcp",
				fmt.Sprintf("%s:%d", host, portNumber),
				0,
				log,
			)
			if err != nil {
				log.Errorf("Error port forwarding %s: %v", port, err)
			}
		}(port)

		forwardedPorts = append(forwardedPorts, port)
	}

	return forwardedPorts, nil
}

func forwardPort(ctx context.Context, containerClient *ssh.Client, port string, exitAfterTimeout time.Duration, log log.Logger) []string {
	parsed, err := nat.ParsePortSpec(port)
	if err != nil {
		log.Debugf("Error parsing appPort %s: %v", port, err)
		return nil
	}

	// try to forward
	forwardedPorts := []string{}
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
			err = devssh.PortForward(ctx, containerClient, "tcp", parsedPort.Binding.HostIP+":"+parsedPort.Binding.HostPort, "tcp", "localhost:"+parsedPort.Port.Port(), exitAfterTimeout, log)
			if err != nil {
				log.Errorf("Error port forwarding %s:%s:%s: %v", parsedPort.Binding.HostIP, parsedPort.Binding.HostPort, parsedPort.Port.Port(), err)
			}
		}(parsedPort)

		forwardedPorts = append(forwardedPorts, parsedPort.Binding.HostPort)
	}

	return forwardedPorts
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
