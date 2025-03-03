package helper

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	command2 "github.com/loft-sh/devpod/pkg/command"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

type SSHGitClone struct {
	KeyFiles []string
	Port     string
}

func NewSSHGitCloneCmd() *cobra.Command {
	cmd := &SSHGitClone{}
	sshCmd := &cobra.Command{
		Use:   "ssh-git-clone",
		Short: "Drop-in ssh replacement in GIT_SSH_COMMAND",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	sshCmd.Flags().StringArrayVar(&cmd.KeyFiles, "key-file", []string{}, "SSH Key file to use")
	sshCmd.Flags().StringVar(&cmd.Port, "port", "22", "SSH port to use, defaults to 22")
	_ = sshCmd.MarkFlagRequired("key-file")
	return sshCmd
}

func (cmd *SSHGitClone) Run(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("expected args in format: {user}@{host} {commands...}, received \"%s\"", strings.Join(args, " "))
	}
	host := args[0]
	sshCmdArgs := args[1:]
	if len(host) == 0 || len(sshCmdArgs) == 0 {
		return fmt.Errorf("unexpected input: host: %s, args: %s", host, strings.Join(sshCmdArgs, " "))
	}

	user, addr, err := parseSSHHost(host)
	if err != nil {
		return err
	}

	sshConfig, err := getConfig(user, cmd.KeyFiles)
	if err != nil {
		return err
	}

	sshClient, err := ssh.Dial("tcp", net.JoinHostPort(addr, cmd.Port), sshConfig)
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
	err = sess.Run(command2.Quote(sshCmdArgs))
	if err != nil {
		return err
	}

	return nil
}

func getConfig(userName string, keyFilePaths []string) (*ssh.ClientConfig, error) {
	signers := []ssh.Signer{}
	for _, keyFilePath := range keyFilePaths {
		out, err := os.ReadFile(keyFilePath)
		if err != nil {
			return nil, fmt.Errorf("read private ssh key: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(out)
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}

		signers = append(signers, signer)
	}

	return &ssh.ClientConfig{
		User:            userName,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signers...)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}, nil
}

func parseSSHHost(host string) (string, string, error) {
	s := strings.SplitN(host, "@", 2)
	if len(s) != 2 {
		return "", "", fmt.Errorf("split host: %s", host)
	}

	return s[0], s[1], nil
}
