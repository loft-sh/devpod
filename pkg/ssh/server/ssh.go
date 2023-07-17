package server

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/log"
	perrors "github.com/pkg/errors"
	"github.com/pkg/sftp"
	"github.com/sirupsen/logrus"
)

var DefaultPort = 8022

func NewServer(addr string, hostKey []byte, keys []ssh.PublicKey, log log.Logger) (*Server, error) {
	shell, err := getShell()
	if err != nil {
		return nil, err
	}

	currentUser, err := user.Current()
	if err != nil {
		return nil, err
	}

	forwardHandler := &ssh.ForwardedTCPHandler{}
	server := &Server{
		shell:       shell,
		log:         log,
		currentUser: currentUser.Username,
		sshServer: ssh.Server{
			Addr: addr,
			LocalPortForwardingCallback: func(ctx ssh.Context, dhost string, dport uint32) bool {
				log.Debugf("Accepted forward: %s:%d", dhost, dport)
				return true
			},
			ReversePortForwardingCallback: func(ctx ssh.Context, host string, port uint32) bool {
				log.Debugf("attempt to bind %s:%d - %s", host, port, "granted")
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
				"sftp": func(s ssh.Session) {
					SftpHandler(s, currentUser.Username, log)
				},
			},
		},
	}

	if len(keys) > 0 {
		server.sshServer.PublicKeyHandler = func(ctx ssh.Context, key ssh.PublicKey) bool {
			for _, k := range keys {
				if ssh.KeysEqual(k, key) {
					return true
				}
			}

			log.Debugf("Declined public key")
			return false
		}
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
	currentUser string
	shell       []string
	sshServer   ssh.Server
	log         log.Logger
}

func getUserShell() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}

	output, err := exec.Command("getent", "passwd", currentUser.Name).Output()
	if err != nil {
		return "", err
	}

	shell := strings.Split(string(output), ":")
	if len(shell) != 7 {
		return "", fmt.Errorf("unexpected getent format: %s", string(output))
	}

	loginShell := strings.TrimSpace(filepath.Base(shell[6]))
	if loginShell == "nologin" {
		return "", fmt.Errorf("no login shell configured")
	}

	return loginShell, nil
}

func getShell() ([]string, error) {
	// try to get a shell
	if runtime.GOOS != "windows" {
		// infere login shell from getent
		shell, err := getUserShell()
		if err == nil {
			return []string{shell}, nil
		}

		// fallback to path discovery if unsuccessful
		_, err = exec.LookPath("bash")
		if err == nil {
			return []string{"bash"}, nil
		}

		_, err = exec.LookPath("sh")
		if err == nil {
			return []string{"sh"}, nil
		}
	}

	// fallback to our in-built shell
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}

	return []string{executable, "helper", "sh"}, nil
}

func (s *Server) handler(sess ssh.Session) {
	ptyReq, winCh, isPty := sess.Pty()
	cmd := s.getCommand(sess, isPty)
	if ssh.AgentRequested(sess) {
		l, err := ssh.NewAgentListener()
		if err != nil {
			s.exitWithError(sess, perrors.Wrap(err, "start agent"))
			return
		}

		defer l.Close()
		go ssh.ForwardAgentConnections(l, sess)
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", "SSH_AUTH_SOCK", l.Addr().String()))
	}

	// start shell session
	var err error
	if isPty {
		s.log.Debugf("Execute SSH server PTY command: %s", strings.Join(cmd.Args, " "))
		err = HandlePTY(sess, ptyReq, winCh, cmd, nil)
	} else {
		s.log.Debugf("Execute SSH server command: %s", strings.Join(cmd.Args, " "))
		err = s.HandleNonPTY(sess, cmd)
	}

	// exit session
	s.exitWithError(sess, err)
}

func (s *Server) HandleNonPTY(sess ssh.Session, cmd *exec.Cmd) (err error) {
	// init pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	// start the command
	err = cmd.Start()
	if err != nil {
		return perrors.Wrap(err, "start command")
	}

	go func() {
		defer stdin.Close()

		_, err := io.Copy(stdin, sess)
		if err != nil {
			s.log.Debugf("Error piping stdin: %v", err)
		}
	}()

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()

		_, err := io.Copy(sess, stdout)
		if err != nil {
			s.log.Debugf("Error piping stdout: %v", err)
		}
	}()

	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()

		_, err := io.Copy(sess.Stderr(), stderr)
		if err != nil {
			s.log.Debugf("Error piping stderr: %v", err)
		}
	}()

	waitGroup.Wait()
	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}

func HandlePTY(sess ssh.Session, ptyReq ssh.Pty, winCh <-chan ssh.Window, cmd *exec.Cmd, decorateReader func(reader io.Reader) io.Reader) (err error) {
	cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
	f, err := startPTY(cmd)
	if err != nil {
		return perrors.Wrap(err, "start pty")
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

func (s *Server) getCommand(sess ssh.Session, isPty bool) *exec.Cmd {
	var cmd *exec.Cmd
	user := sess.User()
	if user == s.currentUser {
		user = ""
	}

	// has user set?
	if user != "" {
		args := []string{}

		// is pty?
		if isPty {
			args = append(args, "-")
		}

		// add user
		args = append(args, sess.User())

		// is there a command?
		if len(sess.RawCommand()) > 0 {
			args = append(args, "-c", sess.RawCommand())
		}

		cmd = exec.Command("su", args...)
	} else {
		args := []string{}
		args = append(args, s.shell[1:]...)
		if isPty {
			args = append(args, "-l")
		}

		if len(sess.RawCommand()) == 0 {
			cmd = exec.Command(s.shell[0], args...)
		} else {
			args = append(args, "-c", sess.RawCommand())
			cmd = exec.Command(s.shell[0], args...)
		}
	}

	// switch default directory
	home, _ := command.GetHome(user)
	cmd.Dir = home
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, sess.Environ()...)
	return cmd
}

func (s *Server) exitWithError(sess ssh.Session, err error) {
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(perrors.Cause(err), &exitError) {
			s.log.Errorf("Exit error: %v", err)
			msg := strings.TrimPrefix(err.Error(), "exec: ")
			if _, err := sess.Stderr().Write([]byte(msg)); err != nil {
				s.log.Errorf("failed to write error to session: %v", err)
			}
		}
	}

	// always exit session
	err = sess.Exit(ExitCode(err))
	if err != nil {
		s.log.Errorf("session failed to exit: %v", err)
	}
}

func SftpHandler(sess ssh.Session, currentUser string, log log.Logger) {
	writer := log.Writer(logrus.DebugLevel, false)
	defer writer.Close()

	user := sess.User()
	if user == currentUser {
		user = ""
	}

	workingDir, _ := command.GetHome(user)
	serverOptions := []sftp.ServerOption{
		sftp.WithDebug(writer),
		sftp.WithServerWorkingDirectory(workingDir),
	}
	server, err := sftp.NewServer(
		sess,
		serverOptions...,
	)
	if err != nil {
		log.Debugf("sftp server init error: %s\n", err)
		return
	}
	defer server.Close()

	// serve
	err = server.Serve()
	if errors.Is(err, io.EOF) {
		_ = sess.Exit(0)
		return
	}

	if err != nil {
		log.Debugf("sftp server completed with error: %v", err)
	}
	_ = sess.Exit(1)
}

func ExitCode(err error) int {
	err = perrors.Cause(err)
	if err == nil {
		return 0
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return 1
	}

	return exitErr.ExitCode()
}

func (s *Server) Serve(listener net.Listener) error {
	return s.sshServer.Serve(listener)
}

func (s *Server) ListenAndServe() error {
	s.log.Debugf("Start ssh server on %s", s.sshServer.Addr)
	return s.sshServer.ListenAndServe()
}
