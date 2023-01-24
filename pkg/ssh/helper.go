package ssh

import (
	"golang.org/x/crypto/ssh"
	"net"
)

func CreateFromConn(conn net.Conn, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return nil, err
	}
	return ssh.NewClient(c, chans, reqs), nil
}
