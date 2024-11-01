package tunnelserver

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/dockercredentials"
	"github.com/loft-sh/devpod/pkg/extract"
	"github.com/loft-sh/devpod/pkg/git"
	"github.com/loft-sh/devpod/pkg/gitcredentials"
	"github.com/loft-sh/devpod/pkg/gitsshsigning"
	"github.com/loft-sh/devpod/pkg/gpg"
	"github.com/loft-sh/devpod/pkg/loftconfig"
	"github.com/loft-sh/devpod/pkg/netstat"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/stdio"
	"github.com/loft-sh/log"
	"github.com/moby/buildkit/frontend/dockerfile/dockerignore"
	perrors "github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func RunServicesServer(ctx context.Context, reader io.Reader, writer io.WriteCloser, allowGitCredentials, allowDockerCredentials bool, forwarder netstat.Forwarder, log log.Logger, options ...Option) error {
	opts := append(options, []Option{
		WithForwarder(forwarder),
		WithAllowGitCredentials(allowGitCredentials),
		WithAllowDockerCredentials(allowDockerCredentials),
	}...)
	tunnelServ := New(log, opts...)

	return tunnelServ.Run(ctx, reader, writer)
}

func RunUpServer(ctx context.Context, reader io.Reader, writer io.WriteCloser, allowGitCredentials, allowDockerCredentials bool, workspace *provider2.Workspace, log log.Logger, options ...Option) (*config.Result, error) {
	opts := append(options, []Option{
		WithWorkspace(workspace),
		WithAllowGitCredentials(allowGitCredentials),
		WithAllowDockerCredentials(allowDockerCredentials),
	}...)
	tunnelServ := New(log, opts...)

	return tunnelServ.RunWithResult(ctx, reader, writer)
}

func RunSetupServer(ctx context.Context, reader io.Reader, writer io.WriteCloser, allowGitCredentials, allowDockerCredentials bool, mounts []*config.Mount, log log.Logger, options ...Option) (*config.Result, error) {
	opts := append(options, []Option{
		WithMounts(mounts),
		WithAllowGitCredentials(allowGitCredentials),
		WithAllowDockerCredentials(allowDockerCredentials),
	}...)
	tunnelServ := New(log, opts...)

	return tunnelServ.RunWithResult(ctx, reader, writer)
}

func New(log log.Logger, options ...Option) *tunnelServer {
	s := &tunnelServer{
		log: log,
	}
	for _, o := range options {
		s = o(s)
	}

	return s
}

type tunnelServer struct {
	tunnel.UnimplementedTunnelServer

	// stream mounts
	mounts []*config.Mount

	forwarder              netstat.Forwarder
	allowGitCredentials    bool
	allowDockerCredentials bool
	result                 *config.Result
	workspace              *provider2.Workspace
	log                    log.Logger
	gitCredentialsOverride gitCredentialsOverride
}

type gitCredentialsOverride struct {
	username string
	token    string
}

func (t *tunnelServer) RunWithResult(ctx context.Context, reader io.Reader, writer io.WriteCloser) (*config.Result, error) {
	lis := stdio.NewStdioListener(reader, writer, false)
	s := grpc.NewServer()
	tunnel.RegisterTunnelServer(s, t)
	reflection.Register(s)
	errChan := make(chan error, 1)
	go func() {
		errChan <- s.Serve(lis)
	}()

	select {
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		return t.result, nil
	}
}

func (t *tunnelServer) Run(ctx context.Context, reader io.Reader, writer io.WriteCloser) error {
	_, err := t.RunWithResult(ctx, reader, writer)
	return err
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
	gitUser, err := gitcredentials.GetUser("")
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

	if t.gitCredentialsOverride.token != "" {
		credentials.Username = t.gitCredentialsOverride.username
		credentials.Password = t.gitCredentialsOverride.token
	} else {
		response, err := gitcredentials.GetCredentials(credentials)
		if err != nil {
			return nil, perrors.Wrap(err, "get git response")
		}
		credentials = response
	}

	out, err := json.Marshal(credentials)
	if err != nil {
		return nil, err
	}

	return &tunnel.Message{Message: string(out)}, nil
}

func (t *tunnelServer) GitSSHSignature(ctx context.Context, message *tunnel.Message) (*tunnel.Message, error) {
	signatureRequest := &gitsshsigning.GitSSHSignatureRequest{}
	err := json.Unmarshal([]byte(message.Message), signatureRequest)
	if err != nil {
		return nil, perrors.Wrap(err, "git ssh sign request")
	}

	signatureResponse, err := signatureRequest.Sign()
	if err != nil {
		return nil, perrors.Wrap(err, "get git ssh signature")
	}

	out, err := json.Marshal(signatureResponse)
	if err != nil {
		return nil, err
	}

	return &tunnel.Message{Message: string(out)}, nil
}

func (t *tunnelServer) LoftConfig(ctx context.Context, message *tunnel.Message) (*tunnel.Message, error) {
	loftConfigRequest := &loftconfig.LoftConfigRequest{}
	err := json.Unmarshal([]byte(message.Message), loftConfigRequest)
	if err != nil {
		return nil, perrors.Wrap(err, "loft platform config request")
	}

	response, err := loftconfig.Read(loftConfigRequest)
	if err != nil {
		return nil, perrors.Wrap(err, "read loft config")
	}

	out, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	return &tunnel.Message{Message: string(out)}, nil
}

func (t *tunnelServer) GPGPublicKeys(ctx context.Context, message *tunnel.Message) (*tunnel.Message, error) {
	rawPubKeys, err := gpg.GetHostPubKey()
	if err != nil {
		return nil, fmt.Errorf("get gpg host public keys: %w", err)
	}

	pubKeyArgument := base64.StdEncoding.EncodeToString(rawPubKeys)

	return &tunnel.Message{Message: pubKeyArgument}, nil
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

func (t *tunnelServer) StreamGitClone(message *tunnel.Empty, stream tunnel.Tunnel_StreamGitCloneServer) error {
	if t.workspace == nil {
		return fmt.Errorf("workspace is nil")
	} else if t.workspace.Source.GitRepository == "" {
		return fmt.Errorf("invalid repository")
	}

	// clone here
	tempDir, err := os.MkdirTemp("", "devpod-git-clone-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// clone repository
	cloneArgs := []string{"clone", t.workspace.Source.GitRepository, tempDir}
	if t.workspace.Source.GitBranch != "" {
		cloneArgs = append(cloneArgs, "--branch", t.workspace.Source.GitBranch)
	}

	// run command
	err = git.CommandContext(context.Background(), cloneArgs...).Run()
	if err != nil {
		return err
	}

	if t.workspace.Source.GitPRReference != "" {
		prBranch := git.GetBranchNameForPR(t.workspace.Source.GitPRReference)

		// git fetch origin pull/996/head:PR996
		fetchArgs := []string{"fetch", "origin", t.workspace.Source.GitPRReference + ":" + prBranch}
		fetchCmd := git.CommandContext(context.Background(), fetchArgs...)
		fetchCmd.Dir = tempDir
		err = fetchCmd.Run()
		if err != nil {
			return err
		}

		// git switch PR996
		switchArgs := []string{"switch", prBranch}
		switchCmd := git.CommandContext(context.Background(), switchArgs...)
		switchCmd.Dir = tempDir
		err = switchCmd.Run()
		if err != nil {
			return err
		}
	} else if t.workspace.Source.GitCommit != "" {
		// reset here
		// git reset --hard $COMMIT_SHA
		resetArgs := []string{"reset", "--hard", t.workspace.Source.GitCommit}
		resetCmd := git.CommandContext(context.Background(), resetArgs...)
		resetCmd.Dir = tempDir

		err = resetCmd.Run()
		if err != nil {
			return err
		}
	}

	buf := bufio.NewWriterSize(NewStreamWriter(stream, t.log), 10*1024)
	err = extract.WriteTar(buf, tempDir, false)
	if err != nil {
		return err
	}

	// make sure buffer is flushed
	return buf.Flush()
}

func (t *tunnelServer) StreamWorkspace(message *tunnel.Empty, stream tunnel.Tunnel_StreamWorkspaceServer) error {
	if t.workspace == nil {
		return fmt.Errorf("workspace is nil")
	}

	// Get .devpodignore files to exclude
	excludes := []string{}
	f, err := os.Open(filepath.Join(t.workspace.Source.LocalFolder, ".devpodignore"))
	if err == nil {
		excludes, err = dockerignore.ReadAll(f)
		if err != nil {
			t.log.Warnf("Error reading .devpodignore file: %v", err)
		}
	}

	buf := bufio.NewWriterSize(NewStreamWriter(stream, t.log), 10*1024)
	err = extract.WriteTarExclude(buf, t.workspace.Source.LocalFolder, false, excludes)
	if err != nil {
		return err
	}

	// make sure buffer is flushed
	return buf.Flush()
}

func (t *tunnelServer) StreamMount(message *tunnel.StreamMountRequest, stream tunnel.Tunnel_StreamMountServer) error {
	var mount *config.Mount
	for _, m := range t.mounts {
		if m.String() == message.Mount {
			mount = m
			break
		}
	}
	if mount == nil {
		return fmt.Errorf("mount %s is not allowed to download", message.Mount)
	}

	// Get .devpodignore files to exclude
	excludes := []string{}
	if t.workspace != nil {
		f, err := os.Open(filepath.Join(t.workspace.Source.LocalFolder, ".devpodignore"))
		if err == nil {
			excludes, err = dockerignore.ReadAll(f)
			if err != nil {
				t.log.Warnf("Error reading .devpodignore file: %v", err)
			}
		}
	}

	buf := bufio.NewWriterSize(NewStreamWriter(stream, t.log), 10*1024)
	err := extract.WriteTarExclude(buf, mount.Source, false, excludes)
	if err != nil {
		return err
	}

	// make sure buffer is flushed
	return buf.Flush()
}
