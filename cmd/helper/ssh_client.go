package helper

import (
	"context"
	"github.com/alessio/shellescape"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"os"
)

type SSHClient struct {
	Address string
	KeyFile string
	User    string
}

// NewSSHClientCmd creates a new ssh command
func NewSSHClientCmd() *cobra.Command {
	cmd := &SSHClient{}
	sshCmd := &cobra.Command{
		Use:   "ssh-client",
		Short: "Starts a new ssh client session",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	sshCmd.Flags().StringVar(&cmd.KeyFile, "key-file", "", "SSH Key file to use")
	sshCmd.Flags().StringVar(&cmd.Address, "address", "", "Address to connect to")
	sshCmd.Flags().StringVar(&cmd.User, "user", "root", "User to connect as")
	_ = sshCmd.MarkFlagRequired("address")
	return sshCmd
}

func (cmd *SSHClient) Run(ctx context.Context, args []string) error {
	sshConfig, err := cmd.getConfig()
	if err != nil {
		return err
	}

	sshClient, err := ssh.Dial("tcp", cmd.Address, sshConfig)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	sess, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	sess.Stdin = os.Stdin
	sess.Stdout = os.Stdout
	sess.Stderr = os.Stderr
	err = sess.Run(shellescape.QuoteCommand(args))
	if err != nil {
		return err
	}

	return nil
}

func (cmd *SSHClient) getConfig() (*ssh.ClientConfig, error) {
	clientConfig := &ssh.ClientConfig{
		User:            cmd.User,
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// key file authentication?
	if cmd.KeyFile != "" {
		out, err := os.ReadFile(cmd.KeyFile)
		if err != nil {
			return nil, errors.Wrap(err, "read private ssh key")
		}

		signer, err := ssh.ParsePrivateKey(out)
		if err != nil {
			return nil, errors.Wrap(err, "parse private key")
		}

		clientConfig.Auth = append(clientConfig.Auth, ssh.PublicKeys(signer))
	}

	return clientConfig, nil
}
