//go:build !windows

package agent

import (
	"os"

	"golang.org/x/crypto/ssh"
	gosshagent "golang.org/x/crypto/ssh/agent"
)

func GetSSHAuthSocket() string {
	sshAuthSocket := os.Getenv("SSH_AUTH_SOCK")
	if sshAuthSocket != "" {
		return sshAuthSocket
	}

	return ""
}

func ForwardToRemote(client *ssh.Client, addr string) error {
	return gosshagent.ForwardToRemote(client, addr)
}

func RequestAgentForwarding(session *ssh.Session) error {
	return gosshagent.RequestAgentForwarding(session)
}
