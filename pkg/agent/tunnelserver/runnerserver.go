package tunnelserver

import (
	"context"
	"fmt"
	"io"

	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	"github.com/loft-sh/devpod/pkg/stdio"
	"github.com/loft-sh/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func RunRunnerServer(ctx context.Context, reader io.Reader, writer io.WriteCloser, allowGitCredentials, allowDockerCredentials bool, gitUsername, gitToken string, log log.Logger) error {
	runnerServ := &runnerServer{
		log:                    log,
		allowGitCredentials:    allowGitCredentials,
		allowDockerCredentials: allowDockerCredentials,
		gitCredentials:         gitCredentialsOverride{username: gitUsername, token: gitToken},
	}

	return runnerServ.Run(ctx, reader, writer)
}

type runnerServer struct {
	tunnel.UnimplementedTunnelServer

	allowGitCredentials    bool
	allowDockerCredentials bool
	log                    log.Logger
	gitCredentials         gitCredentialsOverride
}

func (t *runnerServer) Run(ctx context.Context, reader io.Reader, writer io.WriteCloser) error {
	lis := stdio.NewStdioListener(reader, writer, false)
	s := grpc.NewServer()
	tunnel.RegisterTunnelServer(s, t)
	reflection.Register(s)

	return s.Serve(lis)
}

func (t *runnerServer) DockerCredentials(ctx context.Context, message *tunnel.Message) (*tunnel.Message, error) {
	if !t.allowDockerCredentials {
		return nil, fmt.Errorf("docker credentials forbidden")
	}

	return &tunnel.Message{}, nil
}

func (t *runnerServer) GitCredentials(ctx context.Context, message *tunnel.Message) (*tunnel.Message, error) {
	if !t.allowGitCredentials {
		return nil, fmt.Errorf("git credentials forbidden")
	}

	return &tunnel.Message{}, nil
}

func (t *runnerServer) GPGPublicKeys(ctx context.Context, message *tunnel.Message) (*tunnel.Message, error) {
	return &tunnel.Message{}, nil
}

func (t *runnerServer) GitUser(ctx context.Context, empty *tunnel.Empty) (*tunnel.Message, error) {
	return &tunnel.Message{}, nil
}

func (t *runnerServer) GitSSHSignature(ctx context.Context, message *tunnel.Message) (*tunnel.Message, error) {
	return &tunnel.Message{}, nil
}

func (t *runnerServer) LoftConfig(ctx context.Context, message *tunnel.Message) (*tunnel.Message, error) {
	return &tunnel.Message{}, nil
}

func (t *runnerServer) Ping(context.Context, *tunnel.Empty) (*tunnel.Empty, error) {
	t.log.Debugf("Received ping from agent")
	return &tunnel.Empty{}, nil
}
