package tunnelserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/gitcredentials"
	"github.com/loft-sh/devpod/pkg/stdio"
	"github.com/loft-sh/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func RunProxyServer(ctx context.Context, client tunnel.TunnelClient, reader io.Reader, writer io.WriteCloser, allowGitCredentials, allowDockerCredentials bool, gitUsername, gitToken string, log log.Logger) (*config.Result, error) {
	lis := stdio.NewStdioListener(reader, writer, false)
	s := grpc.NewServer()
	tunnelServ := &proxyServer{
		client: client,
		log:    log,

		gitUsername: gitUsername,
		gitToken:    gitToken,

		allowGitCredentials:    allowGitCredentials,
		allowDockerCredentials: allowDockerCredentials,
	}
	tunnel.RegisterTunnelServer(s, tunnelServ)
	reflection.Register(s)
	errChan := make(chan error, 1)
	go func() {
		errChan <- s.Serve(lis)
	}()

	select {
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		return tunnelServ.result, nil
	}
}

type proxyServer struct {
	tunnel.UnimplementedTunnelServer

	client tunnel.TunnelClient
	result *config.Result
	log    log.Logger

	gitUsername            string
	gitToken               string
	allowGitCredentials    bool
	allowDockerCredentials bool
}

func (t *proxyServer) ForwardPort(ctx context.Context, portRequest *tunnel.ForwardPortRequest) (*tunnel.ForwardPortResponse, error) {
	return t.client.ForwardPort(ctx, portRequest)
}

func (t *proxyServer) StopForwardPort(ctx context.Context, portRequest *tunnel.StopForwardPortRequest) (*tunnel.StopForwardPortResponse, error) {
	return t.client.StopForwardPort(ctx, portRequest)
}

func (t *proxyServer) DockerCredentials(ctx context.Context, message *tunnel.Message) (*tunnel.Message, error) {
	if !t.allowDockerCredentials {
		return nil, fmt.Errorf("docker credentials forbidden")
	}
	return t.client.DockerCredentials(ctx, message)
}

func (t *proxyServer) GitUser(ctx context.Context, empty *tunnel.Empty) (*tunnel.Message, error) {
	return t.client.GitUser(ctx, empty)
}

func (t *proxyServer) GitCredentials(ctx context.Context, message *tunnel.Message) (*tunnel.Message, error) {
	if !t.allowGitCredentials {
		return nil, fmt.Errorf("git credentials forbidden")
	}

	// if we have a git token reuse that and don't ask the user
	if t.gitToken != "" {
		credentials := &gitcredentials.GitCredentials{}
		err := json.Unmarshal([]byte(message.Message), credentials)
		if err != nil {
			return nil, fmt.Errorf("decode git credentials request: %w", err)
		}

		credentials.Password = t.gitToken
		credentials.Username = t.gitUsername

		out, err := json.Marshal(credentials)
		if err != nil {
			return nil, err
		}

		return &tunnel.Message{Message: string(out)}, nil
	}

	return t.client.GitCredentials(ctx, message)
}

func (t *proxyServer) GitSSHSignature(ctx context.Context, message *tunnel.Message) (*tunnel.Message, error) {
	return t.client.GitSSHSignature(ctx, message)
}

func (t *proxyServer) LoftConfig(ctx context.Context, message *tunnel.Message) (*tunnel.Message, error) {
	return t.client.LoftConfig(ctx, message)
}

func (t *proxyServer) GPGPublicKeys(ctx context.Context, message *tunnel.Message) (*tunnel.Message, error) {
	return t.client.GPGPublicKeys(ctx, message)
}

func (t *proxyServer) SendResult(ctx context.Context, result *tunnel.Message) (*tunnel.Empty, error) {
	parsedResult := &config.Result{}
	err := json.Unmarshal([]byte(result.Message), parsedResult)
	if err != nil {
		return nil, err
	}

	t.result = parsedResult
	return t.client.SendResult(ctx, result)
}

func (t *proxyServer) Ping(ctx context.Context, message *tunnel.Empty) (*tunnel.Empty, error) {
	return t.client.Ping(ctx, message)
}

func (t *proxyServer) Log(ctx context.Context, message *tunnel.LogMessage) (*tunnel.Empty, error) {
	return t.client.Log(ctx, message)
}

func (t *proxyServer) StreamGitClone(message *tunnel.Empty, stream tunnel.Tunnel_StreamGitCloneServer) error {
	t.log.Debug("Cloning and reading workspace")
	client, err := t.client.StreamGitClone(context.TODO(), &tunnel.Empty{})
	if err != nil {
		return err
	}

	buf := bufio.NewWriterSize(NewStreamWriter(stream, t.log), 10*1024)
	_, err = io.Copy(buf, NewStreamReader(client, t.log))
	if err != nil {
		return err
	}

	// make sure buffer is flushed
	return buf.Flush()
}

func (t *proxyServer) StreamWorkspace(message *tunnel.Empty, stream tunnel.Tunnel_StreamWorkspaceServer) error {
	t.log.Debug("Start reading workspace")

	client, err := t.client.StreamWorkspace(context.TODO(), &tunnel.Empty{})
	if err != nil {
		return err
	}

	buf := bufio.NewWriterSize(NewStreamWriter(stream, t.log), 10*1024)
	_, err = io.Copy(buf, NewStreamReader(client, t.log))
	if err != nil {
		return err
	}

	// make sure buffer is flushed
	return buf.Flush()
}

func (t *proxyServer) StreamMount(message *tunnel.StreamMountRequest, stream tunnel.Tunnel_StreamMountServer) error {
	t.log.Debug("Start reading mount")

	client, err := t.client.StreamMount(context.TODO(), message)
	if err != nil {
		return err
	}

	buf := bufio.NewWriterSize(NewStreamWriter(stream, t.log), 10*1024)
	_, err = io.Copy(buf, NewStreamReader(client, t.log))
	if err != nil {
		return err
	}

	// make sure buffer is flushed
	return buf.Flush()
}
