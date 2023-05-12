package ssh

import (
	"context"
	"fmt"
	"io"

	"github.com/loft-sh/devpod/pkg/stdio"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

func NewSSHPassClient(user, addr, password string) (*ssh.Client, error) {
	clientConfig := &ssh.ClientConfig{
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	clientConfig.Auth = append(clientConfig.Auth, ssh.Password(password))

	if user != "" {
		clientConfig.User = user
	}

	client, err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("dial to %v failed: %w", addr, err)
	}

	return client, nil
}

func NewSSHClient(user, addr string, keyBytes []byte) (*ssh.Client, error) {
	sshConfig, err := ConfigFromKeyBytes(keyBytes)
	if err != nil {
		return nil, err
	}

	if user != "" {
		sshConfig.User = user
	}

	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("dial to %v failed: %w", addr, err)
	}

	return client, nil
}

func StdioClientFromKeyBytes(keyBytes []byte, reader io.Reader, writer io.WriteCloser, exitOnClose bool) (*ssh.Client, error) {
	return StdioClientFromKeyBytesWithUser(keyBytes, reader, writer, "", exitOnClose)
}

func StdioClientFromKeyBytesWithUser(keyBytes []byte, reader io.Reader, writer io.WriteCloser, user string, exitOnClose bool) (*ssh.Client, error) {
	conn := stdio.NewStdioStream(reader, writer, exitOnClose)
	clientConfig, err := ConfigFromKeyBytes(keyBytes)
	if err != nil {
		return nil, err
	}

	clientConfig.User = user
	c, chans, req, err := ssh.NewClientConn(conn, "stdio", clientConfig)
	if err != nil {
		return nil, err
	}

	return ssh.NewClient(c, chans, req), nil
}

func ConfigFromKeyBytes(keyBytes []byte) (*ssh.ClientConfig, error) {
	clientConfig := &ssh.ClientConfig{
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// key file authentication?
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, errors.Wrap(err, "parse private key")
	}

	clientConfig.Auth = append(clientConfig.Auth, ssh.PublicKeys(signer))
	return clientConfig, nil
}

func Run(ctx context.Context, client *ssh.Client, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	sess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	exit := make(chan struct{})
	defer close(exit)
	go func() {
		select {
		case <-ctx.Done():
			_ = sess.Signal(ssh.SIGINT)
			_ = sess.Close()
		case <-exit:
		}
	}()

	sess.Stdin = stdin
	sess.Stdout = stdout
	sess.Stderr = stderr
	err = sess.Run(command)
	if err != nil {
		return err
	}

	return nil
}
