package ssh

import (
	"context"
	"github.com/loft-sh/devpod/pkg/provider/types"
	"golang.org/x/crypto/ssh"
	"io"
)

func NewSSHRemoteCommandHandler(sshClient *ssh.Client) types.RemoteCommandHandler {
	return &sshHandler{sshClient: sshClient}
}

type sshHandler struct {
	sshClient *ssh.Client
}

func (s *sshHandler) Run(ctx context.Context, cmd string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	sess, err := s.sshClient.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	sess.Stdin = stdin
	sess.Stdout = stdout
	sess.Stderr = stderr
	return sess.Run(cmd)
}

func (s *sshHandler) Close() error {
	return s.sshClient.Close()
}
