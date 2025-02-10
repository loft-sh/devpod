package agent

import (
	"io"
	"os"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	gosshagent "golang.org/x/crypto/ssh/agent"
	"gopkg.in/natefinch/npipe.v2"
)

const (
	channelType      = "auth-agent@openssh.com"
	defaultNamedPipe = "\\\\.\\pipe\\openssh-ssh-agent"
)

/*
 * Cygwin/MSYS2 `SSH_AUTH_SOCK` implementations from ssh-agent(1) are performed using an
 * emulated socket rather than a true AF_UNIX socket. As such, those implementations are
 * incompatible and a user should either utilize the Win32-OpenSSH implementation found
 * in Windows 10/11 or utilize another alternative that support valid AF_UNIX sockets.
 */
func GetSSHAuthSocket() string {
	sshAuthSocket := os.Getenv("SSH_AUTH_SOCK")
	if sshAuthSocket != "" {
		return sshAuthSocket
	}
	if _, err := os.Stat(defaultNamedPipe); err == nil {
		return defaultNamedPipe
	}

	return ""
}

func ForwardToRemote(client *ssh.Client, addr string) error {
	if strings.Contains(addr, "\\\\.\\pipe\\") {
		channels := client.HandleChannelOpen(channelType)
		if channels == nil {
			return errors.New("agent: already have handler for " + channelType)
		}
		conn, err := npipe.Dial(addr)
		if err != nil {
			return err
		}
		conn.Close()

		go func() {
			for ch := range channels {
				channel, reqs, err := ch.Accept()
				if err != nil {
					continue
				}
				go ssh.DiscardRequests(reqs)
				go forwardNamedPipe(channel, addr)
			}
		}()
		return nil
	}
	return gosshagent.ForwardToRemote(client, addr)
}

func RequestAgentForwarding(session *ssh.Session) error {
	return gosshagent.RequestAgentForwarding(session)
}

func forwardNamedPipe(channel ssh.Channel, addr string) {
	conn, err := npipe.Dial(addr)
	if err != nil {
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		io.Copy(conn, channel)
		wg.Done()
	}()
	go func() {
		io.Copy(channel, conn)
		channel.CloseWrite()
		wg.Done()
	}()

	wg.Wait()
	conn.Close()
	channel.Close()
}
