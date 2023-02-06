package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/ssh/server"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"net"
	"os"
	"time"
)

// CommandCmd holds the cmd flags
type CommandCmd struct{}

// NewCommandCmd defines a command
func NewCommandCmd() *cobra.Command {
	cmd := &CommandCmd{}
	commandCmd := &cobra.Command{
		Use:   "command",
		Short: "Command an instance",
		RunE: func(_ *cobra.Command, args []string) error {
			gcloudProvider, err := newProvider(log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(context.Background(), gcloudProvider, provider2.FromEnvironment(), log.Default)
		},
	}

	return commandCmd
}

// Run runs the command logic
func (cmd *CommandCmd) Run(ctx context.Context, provider *gcloudProvider, workspace *provider2.Workspace, log log.Logger) error {
	command := os.Getenv(provider2.CommandEnv)
	if command == "" {
		return fmt.Errorf("command is empty")
	}

	// wait until instance is ready
	name := getName(workspace)
	status, err := getWorkspaceStatus(ctx, name, provider)
	if err != nil {
		return errors.Wrap(err, "get instance status")
	} else if status == nil {
		return fmt.Errorf("instance %s not found", name)
	}

	// check if instance has an external ip
	if len(status.NetworkInterfaces) == 0 || len(status.NetworkInterfaces[0].AccessConfigs) == 0 || status.NetworkInterfaces[0].AccessConfigs[0].NatIP == "" {
		// via cmd
		return cmd.doSSHCommand(ctx, name, provider, command)
	}

	// dial external address
	externalAddress := status.NetworkInterfaces[0].AccessConfigs[0].NatIP
	d := net.Dialer{Timeout: time.Second * 5}
	conn, err := d.Dial("tcp", fmt.Sprintf("%s:%d", externalAddress, server.DefaultPort))
	if err != nil {
		return err
	}

	// get token
	key, err := devssh.GetPrivateKeyRaw(workspace.Context, workspace.ID)
	if err != nil {
		return errors.Wrap(err, "read private key")
	}

	// parse private key
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return errors.Wrap(err, "parse private key")
	}

	// create ssh client
	client, err := devssh.CreateFromConn(conn, name, &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		return errors.Wrap(err, "dial agent")
	}
	defer client.Close()

	sess, err := client.NewSession()
	if err != nil {
		return errors.Wrap(err, "create session")
	}
	defer sess.Close()

	sess.Stdin = os.Stdin
	sess.Stdout = os.Stdout
	sess.Stderr = os.Stderr
	return sess.Run(command)
}

func (cmd *CommandCmd) doSSHCommand(ctx context.Context, name string, provider *gcloudProvider, command string) error {
	args := []string{
		"compute",
		"ssh",
		name,
		"--project=" + provider.Config.Project,
		"--zone=" + provider.Config.Zone,
		"--command",
		command,
	}

	return provider.runCommand(ctx, args, os.Stdout, os.Stderr, os.Stdin)
}
