package container

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	"github.com/loft-sh/devpod/pkg/agent/tunnelserver"
	"github.com/loft-sh/devpod/pkg/credentials"
	"github.com/loft-sh/devpod/pkg/dockercredentials"
	"github.com/loft-sh/devpod/pkg/gitcredentials"
	"github.com/loft-sh/devpod/pkg/gitsshsigning"
	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/devpod/pkg/netstat"
	portpkg "github.com/loft-sh/devpod/pkg/port"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

const ExitCodeIO int = 64

// CredentialsServerCmd holds the cmd flags
type CredentialsServerCmd struct {
	*flags.GlobalFlags

	User string

	ConfigureGitHelper    bool
	ConfigureDockerHelper bool

	ForwardPorts      bool
	GitUserSigningKey string
	Runner            bool
}

// NewCredentialsServerCmd creates a new command
func NewCredentialsServerCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &CredentialsServerCmd{
		GlobalFlags: flags,
	}
	credentialsServerCmd := &cobra.Command{
		Use:   "credentials-server",
		Short: "Starts a credentials server",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			runnerPort, err := credentials.GetRunnerPort()
			if err != nil {
				return err
			}
			if cmd.Runner {
				return cmd.RunRunner(c.Context(), runnerPort)
			}

			port, err := credentials.GetPort()
			if err != nil {
				return err
			}

			return cmd.Run(c.Context(), port, runnerPort)
		},
	}
	credentialsServerCmd.Flags().BoolVar(&cmd.ConfigureGitHelper, "configure-git-helper", false, "If true will configure git helper")
	credentialsServerCmd.Flags().BoolVar(&cmd.ConfigureDockerHelper, "configure-docker-helper", false, "If true will configure docker helper")
	credentialsServerCmd.Flags().BoolVar(&cmd.ForwardPorts, "forward-ports", false, "If true will automatically try to forward open ports within the container")
	credentialsServerCmd.Flags().StringVar(&cmd.GitUserSigningKey, "git-user-signing-key", "", "")
	credentialsServerCmd.Flags().StringVar(&cmd.User, "user", "", "The user to use")
	_ = credentialsServerCmd.MarkFlagRequired("user")
	credentialsServerCmd.Flags().BoolVar(&cmd.Runner, "runner", false, "If true will create a credentials server connected to the runner")

	return credentialsServerCmd
}

// Run runs the command logic
func (cmd *CredentialsServerCmd) Run(ctx context.Context, port int, runnerPort int) error {
	// create a grpc client
	tunnelClient, err := tunnelserver.NewTunnelClient(os.Stdin, os.Stdout, true, ExitCodeIO)
	if err != nil {
		return fmt.Errorf("error creating tunnel client: %w", err)
	}

	// this message serves as a ping to the client
	_, err = tunnelClient.Ping(ctx, &tunnel.Empty{})
	if err != nil {
		return fmt.Errorf("ping client: %w", err)
	}

	// create debug logger
	log := tunnelserver.NewTunnelLogger(ctx, tunnelClient, cmd.Debug)

	// forward ports
	if cmd.ForwardPorts {
		go func() {
			log.Debugf("Start watching & forwarding open ports")
			err = forwardPorts(ctx, tunnelClient, log)
			if err != nil {
				log.Errorf("error forwarding ports: %v", err)
			}
		}()
	}

	addr := net.JoinHostPort("localhost", strconv.Itoa(port))
	if ok, err := portpkg.IsAvailable(addr); !ok || err != nil {
		log.Debugf("Port %d not available, exiting", port)
		return nil
	}

	runnerAddr := checkRunnerCredentialServer(runnerPort)

	// configure docker credential helper
	if cmd.ConfigureDockerHelper && dockerCredentialsAllowed(runnerAddr) {
		err = dockercredentials.ConfigureCredentialsContainer(cmd.User, port, log)
		if err != nil {
			return err
		}
	}

	// configure git user
	err = configureGitUserLocally(ctx, cmd.User, tunnelClient)
	if err != nil {
		log.Debugf("Error configuring git user: %v", err)
		return err
	}

	// configure git credential helper
	if cmd.ConfigureGitHelper && gitCredentialsAllowed(runnerAddr) {
		binaryPath, err := os.Executable()
		if err != nil {
			return err
		}
		err = gitcredentials.ConfigureHelper(binaryPath, cmd.User, port)
		if err != nil {
			return fmt.Errorf("configure git helper: %w", err)
		}

		// cleanup when we are done
		defer func(userName string) {
			_ = gitcredentials.RemoveHelper(userName)
		}(cmd.User)
	}

	// configure git ssh signature helper
	if cmd.GitUserSigningKey != "" {
		decodedKey, err := base64.StdEncoding.DecodeString(cmd.GitUserSigningKey)
		if err != nil {
			return fmt.Errorf("decode git ssh signature key: %w", err)
		}
		err = gitsshsigning.ConfigureHelper(cmd.User, string(decodedKey), log)
		if err != nil {
			return fmt.Errorf("configure git ssh signature helper: %w", err)
		}

		// cleanup when we are done
		defer func(userName string) {
			_ = gitsshsigning.RemoveHelper(userName)
		}(cmd.User)
	}

	return credentials.RunCredentialsServer(ctx, port, tunnelClient, runnerAddr, log)
}

// RunRunner starts the runners credentials server
// It's connected directly to a services server on the runner instead of on the origin developer machine
//
// The origin credentials server (default: port 12049) and the runner credentials server (default: port 12050)
// communicate through https. Since both are connected to their respective peers over stdio, the default mode is
// to always connect external tools (git, docker) to the origin instance. It is then responsible
// for pinging the runners server first.
// The runner will either send a valid response to use, an empty response meaning "no decision" or an error, indicating abortion.
func (cmd *CredentialsServerCmd) RunRunner(ctx context.Context, port int) error {
	// create a grpc client
	tunnelClient, err := tunnelserver.NewTunnelClient(os.Stdin, os.Stdout, true, ExitCodeIO)
	if err != nil {
		return fmt.Errorf("error creating tunnel client: %w", err)
	}

	// this message serves as a ping to the client
	_, err = tunnelClient.Ping(ctx, &tunnel.Empty{})
	if err != nil {
		return fmt.Errorf("ping client: %w", err)
	}

	// create debug logger
	log := tunnelserver.NewTunnelLogger(ctx, tunnelClient, cmd.Debug)

	addr := net.JoinHostPort("localhost", strconv.Itoa(port))
	if ok, err := portpkg.IsAvailable(addr); !ok || err != nil {
		log.Debugf("Port %d not available, exiting", port)
		return nil
	}

	// We go through the same startup procedure the origin credentials server goes through as well
	// This ensures we set up everything according to platform settings if we are in scenarios where we
	// don't have an origin server, for example in web mode.

	if cmd.ConfigureDockerHelper {
		err = dockercredentials.ConfigureCredentialsContainer(cmd.User, port, log)
		if err != nil {
			return err
		}
	}

	err = configureGitUserLocally(ctx, cmd.User, tunnelClient)
	if err != nil {
		log.Debugf("Error configuring git user: %v", err)
	}

	// configure git credential helper
	if cmd.ConfigureGitHelper {
		binaryPath, err := os.Executable()
		if err != nil {
			return err
		}
		err = gitcredentials.ConfigureHelper(binaryPath, cmd.User, port)
		if err != nil {
			return fmt.Errorf("configure git helper: %w", err)
		}

		// cleanup when we are done
		defer func(userName string) {
			_ = gitcredentials.RemoveHelper(userName)
		}(cmd.User)
	}

	// configure git ssh signature helper
	if cmd.GitUserSigningKey != "" {
		err = gitsshsigning.ConfigureHelper(cmd.User, cmd.GitUserSigningKey, log)
		if err != nil {
			return fmt.Errorf("configure git ssh signature helper: %w", err)
		}

		// cleanup when we are done
		defer func(userName string) {
			_ = gitsshsigning.RemoveHelper(userName)
		}(cmd.User)
	}

	return credentials.RunCredentialsServer(ctx, port, tunnelClient, "", log)
}

func configureGitUserLocally(ctx context.Context, userName string, client tunnel.TunnelClient) error {
	// get local credentials
	localGitUser, err := gitcredentials.GetUser(userName)
	if err != nil {
		return err
	} else if localGitUser.Name != "" && localGitUser.Email != "" {
		return nil
	}

	// set user & email if not found
	response, err := client.GitUser(ctx, &tunnel.Empty{})
	if err != nil {
		return fmt.Errorf("retrieve git user: %w", err)
	}

	// parse git user from response
	gitUser := &gitcredentials.GitUser{}
	err = json.Unmarshal([]byte(response.Message), gitUser)
	if err != nil {
		return fmt.Errorf("decode git user: %w", err)
	}

	// don't override what is already there
	if localGitUser.Name != "" {
		gitUser.Name = ""
	}
	if localGitUser.Email != "" {
		gitUser.Email = ""
	}

	// set git user
	err = gitcredentials.SetUser(userName, gitUser)
	if err != nil {
		return fmt.Errorf("set git user & email: %w", err)
	}

	return nil
}

func forwardPorts(ctx context.Context, client tunnel.TunnelClient, log log.Logger) error {
	return netstat.NewWatcher(&forwarder{ctx: ctx, client: client}, log).Run(ctx)
}

type forwarder struct {
	ctx context.Context

	client tunnel.TunnelClient
}

func (f *forwarder) Forward(port string) error {
	_, err := f.client.ForwardPort(f.ctx, &tunnel.ForwardPortRequest{Port: port})
	return err
}

func (f *forwarder) StopForward(port string) error {
	_, err := f.client.StopForwardPort(f.ctx, &tunnel.StopForwardPortRequest{Port: port})
	return err
}

// dockerCredentialsAllowed checks if the runner allows docker credential forwarding
// if we can connect to it
func dockerCredentialsAllowed(runnerAddr string) bool {
	if runnerAddr == "" {
		return true
	}

	rawJSON, err := json.Marshal(&dockercredentials.Request{})
	if err != nil {
		return false
	}
	res, err := devpodhttp.GetHTTPClient().Post(fmt.Sprintf("http://%s/%s", runnerAddr, "docker-credentials"),
		"application/json", bytes.NewReader(rawJSON))

	return res.StatusCode == http.StatusOK && err == nil
}

// gitCredentialsAllowed checks if the runner allows git credential forwarding
// if we can connect to it
func gitCredentialsAllowed(runnerAddr string) bool {
	if runnerAddr == "" {
		return true
	}

	res, err := devpodhttp.GetHTTPClient().Post(fmt.Sprintf("http://%s/%s", runnerAddr, "git-credentials"),
		"application/json", bytes.NewReader([]byte("")))

	return res.StatusCode == http.StatusOK && err == nil
}

// checkRunnerCredentialServer tries to contact the runner credentials server
// and returns it's host:port address if available
func checkRunnerCredentialServer(runnerPort int) string {
	runnerAddr := net.JoinHostPort("localhost", strconv.Itoa(runnerPort))
	runnerAvailable, _ := portpkg.IsAvailable(runnerAddr)
	if runnerAvailable {
		// If the port is free we don't have to check in with runner server
		return ""
	}

	return runnerAddr
}
