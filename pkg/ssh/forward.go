package ssh

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	"github.com/loft-sh/log"
	"golang.org/x/crypto/ssh"
)

func PortForward(
	ctx context.Context,
	client *ssh.Client,
	localNetwork, localAddr, remoteNetwork, remoteAddr string,
	exitAfterTimeout time.Duration,
	log log.Logger,
) error {
	listener, err := net.Listen(localNetwork, localAddr)
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

	counter := newConnectionCounter(ctx, exitAfterTimeout, func() {
		log.Fatal("Stopping devpod up, because it stayed idle for a while. You can disable this via 'devpod context set-options -o EXIT_AFTER_TIMEOUT=false'")
	}, localAddr, log)
	for {
		// waiting for a new connection
		local, err := listener.Accept()
		if err != nil {
			return err
		}

		// tell the counter there is a connection
		counter.Add()

		// forward connection
		go func() {
			defer counter.Dec()

			forward(local, client, remoteNetwork, remoteAddr, log)
		}()
	}
}

func forward(
	localConn net.Conn,
	client *ssh.Client,
	remoteNetwork, remoteAddr string,
	log log.Logger,
) {
	// Setup sshConn (type net.Conn)
	sshConn, err := client.Dial(remoteNetwork, remoteAddr)
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

func ReversePortForward(
	ctx context.Context,
	client *ssh.Client,
	remoteNetwork, remoteAddr, localNetwork, localAddr string,
	exitAfterTimeout time.Duration,
	log log.Logger,
) error {
	listener, err := client.Listen(remoteNetwork, remoteAddr)
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

	counter := newConnectionCounter(ctx, exitAfterTimeout, func() {
		log.Fatal("Stopping devpod up, because it stayed idle for a while. You can disable this via 'devpod context set-options -o EXIT_AFTER_TIMEOUT=false'")
	}, remoteAddr, log)
	for {
		// waiting for a new connection
		remote, err := listener.Accept()
		if err != nil {
			return err
		}

		// tell the counter there is a connection
		counter.Add()

		// forward connection
		go func() {
			defer counter.Dec()

			reverseForward(remote, localNetwork, localAddr, log)
		}()
	}
}

func reverseForward(remoteConn net.Conn, localNetwork, localAddr string, log log.Logger) {
	// Setup localConn (type net.Conn)
	localConn, err := net.Dial(localNetwork, localAddr)
	if err != nil {
		log.Debugf("error dialing remote: %v", err)
		return
	}
	defer localConn.Close()

	// Copy localConn.Reader to sshConn.Writer
	waitGroup := sync.WaitGroup{}
	go func() {
		defer waitGroup.Done()
		defer localConn.Close()

		_, err = io.Copy(localConn, remoteConn)
		if err != nil {
			log.Debugf("error copying to local: %v", err)
		}
	}()
	waitGroup.Add(1)

	// Copy sshConn.Reader to localConn.Writer
	go func() {
		defer waitGroup.Done()
		defer remoteConn.Close()

		_, err = io.Copy(remoteConn, localConn)
		if err != nil {
			log.Debugf("error copying to remote: %v", err)
		}
	}()
	waitGroup.Add(1)
	waitGroup.Wait()
}
