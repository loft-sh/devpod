package container

import (
	"context"
	"fmt"
	"os"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	"github.com/loft-sh/devpod/pkg/agent/tunnelserver"
	"github.com/loft-sh/devpod/pkg/credentials"
	"github.com/loft-sh/devpod/pkg/netstat"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// CredentialsServerCmd holds the cmd flags
type CredentialsServerCmd struct {
	*flags.GlobalFlags

	User string

	ConfigureGitHelper    bool
	ConfigureDockerHelper bool

	ForwardPorts bool
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
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}
	credentialsServerCmd.Flags().BoolVar(&cmd.ConfigureGitHelper, "configure-git-helper", false, "If true will configure git helper")
	credentialsServerCmd.Flags().BoolVar(&cmd.ConfigureDockerHelper, "configure-docker-helper", false, "If true will configure docker helper")
	credentialsServerCmd.Flags().BoolVar(&cmd.ForwardPorts, "forward-ports", false, "If true will automatically try to forward open ports within the container")
	credentialsServerCmd.Flags().StringVar(&cmd.User, "user", "", "The user to use")
	_ = credentialsServerCmd.MarkFlagRequired("user")
	return credentialsServerCmd
}

// Run runs the command logic
func (cmd *CredentialsServerCmd) Run(ctx context.Context, _ []string) error {
	// create a grpc client
	tunnelClient, err := tunnelserver.NewTunnelClient(os.Stdin, os.Stdout, true)
	if err != nil {
		return fmt.Errorf("error creating tunnel client: %w", err)
	}

	// this message serves as a ping to the client
	_, err = tunnelClient.Ping(ctx, &tunnel.Empty{})
	if err != nil {
		return errors.Wrap(err, "ping client")
	}

	// create debug logger
	log := tunnelserver.NewTunnelLogger(ctx, tunnelClient, cmd.Debug)
	log.Debugf("Start credentials server")

	port, err := credentials.GetPort()
	if err != nil {
		return err
	}

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

	// run the credentials server
	return credentials.RunCredentialsServer(ctx, cmd.User, port, true, cmd.ConfigureGitHelper, cmd.ConfigureDockerHelper, tunnelClient, log)
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
