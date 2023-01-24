package server

import (
	"fmt"
	"github.com/gliderlabs/ssh"
	"github.com/loft-sh/devpod/pkg/ssh/server/stderrlog"
	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

var DefaultPort = 8022

func NewServer(addr string, hostKey []byte, keys []ssh.PublicKey) (*Server, error) {
	shell, err := getShell()
	if err != nil {
		return nil, err
	}

	forwardHandler := &ssh.ForwardedTCPHandler{}
	server := &Server{
		shell: shell,
		sshServer: ssh.Server{
			Addr: addr,
			PublicKeyHandler: func(ctx ssh.Context, key ssh.PublicKey) bool {
				if len(keys) == 0 {
					return true
				}

				for _, k := range keys {
					if ssh.KeysEqual(k, key) {
						return true
					}
				}

				stderrlog.Debugf("Declined public key")
				return false
			},
			LocalPortForwardingCallback: func(ctx ssh.Context, dhost string, dport uint32) bool {
				stderrlog.Debugf("Accepted forward", dhost, dport)
				return true
			},
			ReversePortForwardingCallback: func(ctx ssh.Context, host string, port uint32) bool {
				stderrlog.Debugf("attempt to bind", host, port, "granted")
				return true
			},
			ChannelHandlers: map[string]ssh.ChannelHandler{
				"direct-tcpip": ssh.DirectTCPIPHandler,
				"session":      ssh.DefaultSessionHandler,
			},
			RequestHandlers: map[string]ssh.RequestHandler{
				"tcpip-forward":        forwardHandler.HandleSSHRequest,
				"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
			},
			SubsystemHandlers: map[string]ssh.SubsystemHandler{
				"sftp": SftpHandler,
			},
		},
	}

	if len(hostKey) > 0 {
		err = server.sshServer.SetOption(ssh.HostKeyPEM(hostKey))
		if err != nil {
			return nil, err
		}
	}

	server.sshServer.Handler = server.handler
	return server, nil
}

type Server struct {
	shell     string
	sshServer ssh.Server
}

func getShell() (string, error) {
	// try to get a shell
	_, err := exec.LookPath("bash")
	if err != nil {
		_, err := exec.LookPath("sh")
		if err != nil {
			return "", fmt.Errorf("neither 'bash' nor 'sh' found in container. Please make sure at least one is available in the container $PATH")
		}

		return "sh", nil
	}

	return "bash", nil
}

func (s *Server) handler(sess ssh.Session) {
	cmd := s.getCommand(sess)
	if ssh.AgentRequested(sess) {
		l, err := ssh.NewAgentListener()
		if err != nil {
			exitWithError(sess, errors.Wrap(err, "start agent"))
			return
		}

		defer l.Close()
		go ssh.ForwardAgentConnections(l, sess)
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", "SSH_AUTH_SOCK", l.Addr().String()))
	}

	// start shell session
	var err error
	ptyReq, winCh, isPty := sess.Pty()
	if isPty {
		err = HandlePTY(sess, ptyReq, winCh, cmd, nil)
	} else {
		err = HandleNonPTY(sess, cmd, nil)
	}

	// exit session
	exitWithError(sess, err)
}

func HandleNonPTY(sess ssh.Session, cmd *exec.Cmd, decorateReader func(reader io.Reader) io.Reader) (err error) {
	// init pipes
	stdinWriter, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdoutReader, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderrReader, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	// start the command
	err = cmd.Start()
	if err != nil {
		return errors.Wrap(err, "start command")
	}

	stdoutDone := make(chan struct{})
	go func() {
		defer close(stdoutDone)
		defer stdoutReader.Close()

		var reader io.Reader = stdoutReader
		if decorateReader != nil {
			reader = decorateReader(stdoutReader)
		}

		_, _ = io.Copy(sess, reader)
	}()

	stderrDone := make(chan struct{})
	go func() {
		defer close(stderrDone)
		defer stderrReader.Close()

		var reader io.Reader = stderrReader
		if decorateReader != nil {
			reader = decorateReader(stderrReader)
		}

		_, _ = io.Copy(sess.Stderr(), reader)
	}()

	go func() {
		defer stdinWriter.Close()

		_, _ = io.Copy(stdinWriter, sess)
	}()

	err = cmd.Wait()
	if err != nil {
		return err
	}

	// make sure channels are closed
	select {
	case <-stdoutDone:
		select {
		case <-stderrDone:
		case <-time.After(time.Second):
		}
	case <-time.After(time.Second):
	}

	return nil
}

func HandlePTY(sess ssh.Session, ptyReq ssh.Pty, winCh <-chan ssh.Window, cmd *exec.Cmd, decorateReader func(reader io.Reader) io.Reader) (err error) {
	cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
	f, err := startPTY(cmd)
	if err != nil {
		return errors.Wrap(err, "start pty")
	}
	defer f.Close()

	go func() {
		for win := range winCh {
			setWinSize(f, win.Width, win.Height)
		}
	}()

	go func() {
		defer f.Close()

		// copy stdin
		_, _ = io.Copy(f, sess)
	}()

	stdoutDoneChan := make(chan struct{})
	go func() {
		defer f.Close()
		defer close(stdoutDoneChan)

		var reader io.Reader = f
		if decorateReader != nil {
			reader = decorateReader(f)
		}

		// copy stdout
		_, _ = io.Copy(sess, reader)
	}()

	err = cmd.Wait()
	if err != nil {
		return err
	}

	select {
	case <-stdoutDoneChan:
	case <-time.After(time.Second):
	}
	return nil
}

func (s *Server) getCommand(sess ssh.Session) *exec.Cmd {
	var cmd *exec.Cmd
	if len(sess.RawCommand()) == 0 {
		cmd = exec.Command(s.shell)
	} else {
		args := []string{"-c", sess.RawCommand()}
		cmd = exec.Command(s.shell, args...)
	}

	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, sess.Environ()...)

	setUser(sess.User(), cmd)
	return cmd
}

func exitWithError(s ssh.Session, err error) {
	if err != nil {
		stderrlog.Errorf("%v", err)
		msg := strings.TrimPrefix(err.Error(), "exec: ")
		if _, err := s.Stderr().Write([]byte(msg)); err != nil {
			stderrlog.Errorf("failed to write error to session: %v", err)
		}
	}

	// always exit session
	err = s.Exit(ExitCode(err))
	if err != nil {
		stderrlog.Errorf("session failed to exit: %v", err)
	}
}

func SftpHandler(sess ssh.Session) {
	debugStream := io.Discard
	serverOptions := []sftp.ServerOption{
		sftp.WithDebug(debugStream),
	}
	server, err := sftp.NewServer(
		sess,
		serverOptions...,
	)
	if err != nil {
		log.Printf("sftp server init error: %s\n", err)
		return
	}
	if err := server.Serve(); err == io.EOF {
		server.Close()
		fmt.Println("sftp client exited session.")
	} else if err != nil {
		fmt.Println("sftp server completed with error:", err)
	}
}

func ExitCode(err error) int {
	err = errors.Cause(err)
	if err == nil {
		return 0
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return 1
	}

	return exitErr.ExitCode()
}

func (s *Server) Serve(listener net.Listener) error {
	return s.sshServer.Serve(listener)
}

func (s *Server) ListenAndServe() error {
	stderrlog.Infof("Start ssh server on %s", s.sshServer.Addr)
	return s.sshServer.ListenAndServe()
}
