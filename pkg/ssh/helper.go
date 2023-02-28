package ssh

import (
	"bytes"
	"github.com/loft-sh/devpod/pkg/stdio"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"io"
)

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

func Output(client *ssh.Client, command string) ([]byte, []byte, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := Run(client, command, nil, stdout, stderr)
	return stdout.Bytes(), stderr.Bytes(), err
}

func CombinedOutput(client *ssh.Client, command string) ([]byte, error) {
	buf := &bytes.Buffer{}
	err := Run(client, command, nil, buf, buf)
	return buf.Bytes(), err
}

func Run(client *ssh.Client, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	sess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	sess.Stdin = stdin
	sess.Stdout = stdout
	sess.Stderr = stderr
	err = sess.Run(command)
	if err != nil {
		return err
	}

	return nil
}
