package helper

import (
	"encoding/base64"
	"fmt"
	"github.com/gliderlabs/ssh"
	helperssh "github.com/loft-sh/devpod/pkg/ssh/server"
	"github.com/loft-sh/devpod/pkg/ssh/server/port"
	"github.com/loft-sh/devpod/pkg/ssh/server/stderrlog"
	"github.com/loft-sh/devpod/pkg/stdio"
	"github.com/loft-sh/devpod/pkg/token"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
)

// SSHServerCmd holds the ssh server cmd flags
type SSHServerCmd struct {
	Token   string
	Address string
	Stdio   bool
}

// NewSSHServerCmd creates a new ssh command
func NewSSHServerCmd() *cobra.Command {
	cmd := &SSHServerCmd{}
	sshCmd := &cobra.Command{
		Use:   "ssh-server",
		Short: "Starts a new ssh server",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}

	sshCmd.Flags().StringVar(&cmd.Address, "address", fmt.Sprintf("0.0.0.0:%d", helperssh.DefaultPort), "Address to listen to")
	sshCmd.Flags().BoolVar(&cmd.Stdio, "stdio", false, "Will listen on stdout and stdin instead of an address")
	sshCmd.Flags().StringVar(&cmd.Token, "token", "", "Base64 encoded token to use")
	return sshCmd
}

// Run runs the command logic
func (cmd *SSHServerCmd) Run(_ *cobra.Command, _ []string) error {
	if cmd.Token == "" {
		return fmt.Errorf("token is missing")
	}

	t, err := token.ParseToken(cmd.Token)
	if err != nil {
		return errors.Wrap(err, "parse token")
	}

	var keys []ssh.PublicKey
	if t.AuthorizedKeys != "" {
		keyBytes, err := base64.StdEncoding.DecodeString(t.AuthorizedKeys)
		if err != nil {
			return fmt.Errorf("seems like the provided encoded string is not base64 encoded")
		}

		for len(keyBytes) > 0 {
			key, _, _, rest, err := ssh.ParseAuthorizedKey(keyBytes)
			if err != nil {
				return errors.Wrap(err, "parse authorized key")
			}

			keys = append(keys, key)
			keyBytes = rest
		}
	}

	hostKey := []byte{}
	if len(t.HostKey) > 0 {
		var err error
		hostKey, err = base64.StdEncoding.DecodeString(t.HostKey)
		if err != nil {
			return fmt.Errorf("decode host key")
		}
	}

	server, err := helperssh.NewServer(cmd.Address, hostKey, keys)
	if err != nil {
		return err
	}

	// should we listen on stdout & stdin?
	if cmd.Stdio {
		lis := stdio.NewStdioListener(os.Stdin, os.Stdout, true)
		return server.Serve(lis)
	}

	// check if ssh is already running at that port
	available, err := port.IsAvailable(cmd.Address)
	if !available {
		if err != nil {
			return fmt.Errorf("address %s already in use: %v", cmd.Address, err)
		}

		stderrlog.Debugf("address %s already in use", cmd.Address)
		return nil
	}

	return server.ListenAndServe()
}
