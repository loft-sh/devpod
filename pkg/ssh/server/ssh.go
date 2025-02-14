package server

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"

	"github.com/loft-sh/devpod/pkg/shell"
	"github.com/loft-sh/log"
	"github.com/loft-sh/ssh"
)

const (
	DefaultPort     int = 8022
	DefaultUserPort int = 12023
)

type Server interface {
	Serve(listener net.Listener) error
	ListenAndServe() error
}

type server struct {
	currentUser string
	shell       []string
	workdir     string
	reuseSock   string
	sshServer   ssh.Server
	log         log.Logger
}

func NewServer(addr string, hostKey []byte, keys []ssh.PublicKey, workdir string, reuseSock string, log log.Logger) (Server, error) {
	sh, err := shell.GetShell("")
	if err != nil {
		return nil, err
	}

	currentUser, err := user.Current()
	if err != nil {
		return nil, err
	}

	forwardHandler := &ssh.ForwardedTCPHandler{}
	forwardedUnixHandler := &ssh.ForwardedUnixHandler{}
	server := &server{
		shell:       sh,
		workdir:     workdir,
		reuseSock:   reuseSock,
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
			ReverseUnixForwardingCallback: func(ctx ssh.Context, socketPath string) bool {
				log.Debugf("attempt to bind socket %s", socketPath)

				_, err := os.Stat(socketPath)
				if err == nil {
					log.Debugf("%s already exists, removing", socketPath)

					_ = os.Remove(socketPath)
				}

				return true
			},
			ChannelHandlers: map[string]ssh.ChannelHandler{
				"direct-tcpip":                   ssh.DirectTCPIPHandler,
				"direct-streamlocal@openssh.com": ssh.DirectStreamLocalHandler,
				"session":                        ssh.DefaultSessionHandler,
			},
			RequestHandlers: map[string]ssh.RequestHandler{
				"tcpip-forward":                          forwardHandler.HandleSSHRequest,
				"streamlocal-forward@openssh.com":        forwardedUnixHandler.HandleSSHRequest,
				"cancel-streamlocal-forward@openssh.com": forwardedUnixHandler.HandleSSHRequest,
				"cancel-tcpip-forward":                   forwardHandler.HandleSSHRequest,
			},
			SubsystemHandlers: map[string]ssh.SubsystemHandler{
				"sftp": func(s ssh.Session) {
					sftpHandler(s, currentUser.Username, log)
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

func (s *server) handler(sess ssh.Session) {
	var err error
	ptyReq, winCh, isPty := sess.Pty()
	cmd := s.getCommand(sess, isPty)

	if ssh.AgentRequested(sess) {
		l, tmpDir, err := setupAgentListener(sess, s.reuseSock)
		if err != nil {
			exitWithError(sess, err, s.log)
			return
		}
		defer l.Close()
		defer os.RemoveAll(tmpDir)

		go ssh.ForwardAgentConnections(l, sess)

		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", "SSH_AUTH_SOCK", l.Addr().String()))
	}

	// start shell session
	if isPty {
		err = execPTY(sess, ptyReq, winCh, cmd, s.log)
	} else {
		err = execNonPTY(sess, cmd, s.log)
	}

	// exit session
	exitWithError(sess, err, s.log)
}

func (s *server) getCommand(sess ssh.Session, isPty bool) *exec.Cmd {
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

	cmd.Dir = findWorkdir(s.workdir, user)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, sess.Environ()...)
	return cmd
}

func (s *server) Serve(listener net.Listener) error {
	return s.sshServer.Serve(listener)
}

func (s *server) ListenAndServe() error {
	s.log.Debugf("Start ssh server on %s", s.sshServer.Addr)
	return s.sshServer.ListenAndServe()
}
