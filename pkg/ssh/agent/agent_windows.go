package agent

import (
	"io"
	"os"
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
