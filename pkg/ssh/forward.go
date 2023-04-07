package ssh

import (
	"context"
	"io"
	"net"
	"sync"

	"github.com/loft-sh/devpod/pkg/log"
	"golang.org/x/crypto/ssh"
)

func PortForward(ctx context.Context, client *ssh.Client, localAddr, remoteAddr string, log log.Logger) error {
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return err
	}
	defer listener.Close()

	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-done:
		case <-ctx.Done():
			_ = listener.Close()
		}
	}()

	for {
		// waiting for a new connection
		local, err := listener.Accept()
		if err != nil {
			return err
		}

		// forward connection
		go forward(local, client, remoteAddr, log)
	}
}

func forward(localConn net.Conn, client *ssh.Client, remoteAddr string, log log.Logger) {
	// Setup sshConn (type net.Conn)
	sshConn, err := client.Dial("tcp", remoteAddr)
	if err != nil {
		log.Debugf("error dialing remote: %v", err)
		return
	}
	defer sshConn.Close()

	// Copy localConn.Reader to sshConn.Writer
	waitGroup := sync.WaitGroup{}
	go func() {
		defer waitGroup.Done()
		defer sshConn.Close()

		_, err = io.Copy(sshConn, localConn)
		if err != nil {
			log.Debugf("error copying to remote: %v", err)
		}
	}()
	waitGroup.Add(1)

	// Copy sshConn.Reader to localConn.Writer
	go func() {
		defer waitGroup.Done()
		defer localConn.Close()

		_, err = io.Copy(localConn, sshConn)
		if err != nil {
			log.Debugf("error copying to local: %v", err)
		}
	}()
	waitGroup.Add(1)
	waitGroup.Wait()
}
