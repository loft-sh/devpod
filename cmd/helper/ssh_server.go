package helper

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	helperssh "github.com/loft-sh/devpod/pkg/ssh/server"
	"github.com/loft-sh/devpod/pkg/ssh/server/port"
	"github.com/loft-sh/devpod/pkg/stdio"
	"github.com/loft-sh/devpod/pkg/token"
	"github.com/loft-sh/log"
	"github.com/loft-sh/ssh"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// SSHServerCmd holds the ssh server cmd flags
type SSHServerCmd struct {
	*flags.GlobalFlags

	Token            string
	Address          string
	Stdio            bool
	TrackActivity    bool
	ReuseSSHAuthSock string
	Workdir          string
}

// NewSSHServerCmd creates a new ssh command
func NewSSHServerCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &SSHServerCmd{
		GlobalFlags: flags,
	}
	sshCmd := &cobra.Command{
		Use:   "ssh-server",
		Short: "Starts a new ssh server",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}

	sshCmd.Flags().StringVar(&cmd.Address, "address", fmt.Sprintf("0.0.0.0:%d", helperssh.DefaultPort), "Address to listen to")
	sshCmd.Flags().BoolVar(&cmd.Stdio, "stdio", false, "Will listen on stdout and stdin instead of an address")
	sshCmd.Flags().BoolVar(&cmd.TrackActivity, "track-activity", false, "If enabled will write the last activity time to a file")
	sshCmd.Flags().StringVar(&cmd.ReuseSSHAuthSock, "reuse-ssh-auth-sock", "", "If set, the SSH_AUTH_SOCK is expected to already be available in the workspace (under /tmp using the key provided) and the connection reuses this instead of creating a new one")
	_ = sshCmd.Flags().MarkHidden("reuse-ssh-auth-sock")
	sshCmd.Flags().StringVar(&cmd.Token, "token", "", "Base64 encoded token to use")
	sshCmd.Flags().StringVar(&cmd.Workdir, "workdir", "", "Directory where commands will run on the host")
	return sshCmd
}

// Run runs the command logic
func (cmd *SSHServerCmd) Run(_ *cobra.Command, _ []string) error {
	var (
		keys    []ssh.PublicKey
		hostKey []byte
		err     error
	)
	if cmd.Token != "" {
		// parse token
		t, err := token.ParseToken(cmd.Token)
		if err != nil {
			return errors.Wrap(err, "parse token")
		}

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

		if len(t.HostKey) > 0 {
			hostKey, err = base64.StdEncoding.DecodeString(t.HostKey)
			if err != nil {
				return fmt.Errorf("decode host key")
			}
		}
	}

	// start the server
	server, err := helperssh.NewServer(cmd.Address, hostKey, keys, cmd.Workdir, cmd.ReuseSSHAuthSock, log.Default.ErrorStreamOnly())
	if err != nil {
		return err
	}

	// should we listen on stdout & stdin?
	if cmd.Stdio {
		if cmd.TrackActivity {
			go func() {
				_, err = os.Stat(agent.ContainerActivityFile)
				if err != nil {
					err = os.WriteFile(agent.ContainerActivityFile, nil, 0o777)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
						return
					}

					_ = os.Chmod(agent.ContainerActivityFile, 0o777)
				}

				for {
					time.Sleep(time.Second * 10)
					file, _ := os.Create(agent.ContainerActivityFile)
					file.Close()
				}
			}()
		}

		lis := stdio.NewStdioListener(os.Stdin, os.Stdout, true)
		return server.Serve(lis)
	}

	// check if ssh is already running at that port
	available, err := port.IsAvailable(cmd.Address)
	if !available {
		if err != nil {
			return fmt.Errorf("address %s already in use: %w", cmd.Address, err)
		}

		log.Default.ErrorStreamOnly().Infof("address %s already in use", cmd.Address)
		return nil
	}

	return server.ListenAndServe()
}
