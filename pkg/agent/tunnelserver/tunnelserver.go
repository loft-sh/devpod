package tunnelserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/dockercredentials"
	"github.com/loft-sh/devpod/pkg/extract"
	"github.com/loft-sh/devpod/pkg/git"
	"github.com/loft-sh/devpod/pkg/gitcredentials"
	"github.com/loft-sh/devpod/pkg/netstat"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/stdio"
	"github.com/loft-sh/log"
	perrors "github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func RunTunnelServer(ctx context.Context, reader io.Reader, writer io.WriteCloser, allowGitCredentials, allowDockerCredentials bool, workspace *provider2.Workspace, forwarder netstat.Forwarder, log log.Logger) (*config.Result, error) {
	lis := stdio.NewStdioListener(reader, writer, false)
	s := grpc.NewServer()
	tunnelServ := &tunnelServer{
		workspace:              workspace,
		forwarder:              forwarder,
		allowGitCredentials:    allowGitCredentials,
		allowDockerCredentials: allowDockerCredentials,
		log:                    log,
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

type tunnelServer struct {
	tunnel.UnimplementedTunnelServer

	forwarder              netstat.Forwarder
	allowGitCredentials    bool
	allowDockerCredentials bool
	result                 *config.Result
	workspace              *provider2.Workspace
	log                    log.Logger
}

func (t *tunnelServer) ForwardPort(ctx context.Context, portRequest *tunnel.ForwardPortRequest) (*tunnel.ForwardPortResponse, error) {
	if t.forwarder == nil {
		return nil, fmt.Errorf("cannot forward ports")
	}

	err := t.forwarder.Forward(portRequest.Port)
	if err != nil {
		return nil, fmt.Errorf("error forwarding port %s: %w", portRequest.Port, err)
	}

	return &tunnel.ForwardPortResponse{}, nil
}
func (t *tunnelServer) StopForwardPort(ctx context.Context, portRequest *tunnel.StopForwardPortRequest) (*tunnel.StopForwardPortResponse, error) {
	if t.forwarder == nil {
		return nil, fmt.Errorf("cannot forward ports")
	}

	err := t.forwarder.StopForward(portRequest.Port)
	if err != nil {
		return nil, fmt.Errorf("error stop forwarding port %s: %w", portRequest.Port, err)
	}

	return &tunnel.StopForwardPortResponse{}, nil
}

func (t *tunnelServer) DockerCredentials(ctx context.Context, message *tunnel.Message) (*tunnel.Message, error) {
	if !t.allowDockerCredentials {
		return nil, fmt.Errorf("docker credentials forbidden")
	}

	request := &dockercredentials.Request{}
	err := json.Unmarshal([]byte(message.Message), request)
	if err != nil {
		return nil, err
	}

	// check if list or get
	if request.ServerURL != "" {
		credentials, err := dockercredentials.GetAuthConfig(request.ServerURL)
		if err != nil {
			return nil, err
		}

		out, err := json.Marshal(credentials)
		if err != nil {
			return nil, err
		}

		return &tunnel.Message{Message: string(out)}, nil
	}

	// do a list
	listResponse, err := dockercredentials.ListCredentials()
	if err != nil {
		return nil, err
	}

	out, err := json.Marshal(listResponse)
	if err != nil {
		return nil, err
	}

	return &tunnel.Message{Message: string(out)}, nil
}

func (t *tunnelServer) GitUser(ctx context.Context, empty *tunnel.Empty) (*tunnel.Message, error) {
	gitUser, err := gitcredentials.GetUser()
	if err != nil {
		return nil, err
	}

	out, err := json.Marshal(gitUser)
	if err != nil {
		return nil, err
	}

	return &tunnel.Message{
		Message: string(out),
	}, nil
}

func (t *tunnelServer) GitCredentials(ctx context.Context, message *tunnel.Message) (*tunnel.Message, error) {
	if !t.allowGitCredentials {
		return nil, fmt.Errorf("git credentials forbidden")
	}

	credentials := &gitcredentials.GitCredentials{}
	err := json.Unmarshal([]byte(message.Message), credentials)
	if err != nil {
		return nil, perrors.Wrap(err, "decode git credentials request")
	}

	response, err := gitcredentials.GetCredentials(credentials)
	if err != nil {
		return nil, perrors.Wrap(err, "get git response")
	}

	out, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	return &tunnel.Message{Message: string(out)}, nil
}

func (t *tunnelServer) SendResult(ctx context.Context, result *tunnel.Message) (*tunnel.Empty, error) {
	parsedResult := &config.Result{}
	err := json.Unmarshal([]byte(result.Message), parsedResult)
	if err != nil {
		return nil, err
	}

	t.result = parsedResult
	return &tunnel.Empty{}, nil
}

func (t *tunnelServer) Ping(context.Context, *tunnel.Empty) (*tunnel.Empty, error) {
	t.log.Debugf("Received ping from agent")
	return &tunnel.Empty{}, nil
}

func (t *tunnelServer) Log(ctx context.Context, message *tunnel.LogMessage) (*tunnel.Empty, error) {
	if message.LogLevel == tunnel.LogLevel_DEBUG {
		t.log.Debug(strings.TrimSpace(message.Message))
	} else if message.LogLevel == tunnel.LogLevel_INFO {
		t.log.Info(strings.TrimSpace(message.Message))
	} else if message.LogLevel == tunnel.LogLevel_WARNING {
		t.log.Warn(strings.TrimSpace(message.Message))
	} else if message.LogLevel == tunnel.LogLevel_ERROR {
		t.log.Error(strings.TrimSpace(message.Message))
	} else if message.LogLevel == tunnel.LogLevel_DONE {
		t.log.Done(strings.TrimSpace(message.Message))
	}

	return &tunnel.Empty{}, nil
}

func (t *tunnelServer) GitCloneAndRead(response *tunnel.Empty, stream tunnel.Tunnel_GitCloneAndReadServer) error {
	if t.workspace == nil {
		return fmt.Errorf("workspace is nil")
	}

	if t.workspace.Source.GitRepository == "" {
		return fmt.Errorf("invalid repository")
	}

	gitCloneDir := filepath.Join(t.workspace.Folder, "source")

	// clone here
	// git clone --bare --depth=1 $REPO
	cloneArgs := []string{"clone", t.workspace.Source.GitRepository, gitCloneDir}
	if t.workspace.Source.GitBranch != "" {
		cloneArgs = append(cloneArgs, "--branch", t.workspace.Source.GitBranch)
	}

	err := git.CommandContext(context.Background(), cloneArgs...).Run()
	if err != nil {
		return err
	}

	if t.workspace.Source.GitCommit != "" {
		// reset here
		// git reset --hard $COMMIT_SHA
		resetArgs := []string{"reset", "--hard", t.workspace.Source.GitCommit}
		resetCmd := git.CommandContext(context.Background(), resetArgs...)
		resetCmd.Dir = gitCloneDir

		err = resetCmd.Run()
		if err != nil {
			return err
		}
	}

	buf := bufio.NewWriterSize(NewStreamWriter(stream, t.log), 10*1024)
	err = extract.WriteTar(buf, gitCloneDir, false)
	if err != nil {
		return err
	}

	// make sure buffer is flushed
	return buf.Flush()
}

func (t *tunnelServer) ReadWorkspace(response *tunnel.Empty, stream tunnel.Tunnel_ReadWorkspaceServer) error {
	if t.workspace == nil {
		return fmt.Errorf("workspace is nil")
	}

	buf := bufio.NewWriterSize(NewStreamWriter(stream, t.log), 10*1024)
	err := extract.WriteTar(buf, t.workspace.Source.LocalFolder, false)
	if err != nil {
		return err
	}

	// make sure buffer is flushed
	return buf.Flush()
}
