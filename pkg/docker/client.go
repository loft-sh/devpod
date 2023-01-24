package docker

import (
	"context"
	"github.com/docker/cli/cli"
	"github.com/docker/docker/api/types"
	"github.com/loft-sh/devpod/pkg/log"
	"io"

	dockerclient "github.com/docker/docker/client"
	"github.com/pkg/errors"
)

// Client is a client for docker
type Client struct {
	dockerclient.CommonAPIClient
}

// NewClient creates a new docker client
func NewClient(ctx context.Context, log log.Logger) (*Client, error) {
	cli, err := newDockerClientFromEnvironment()
	if err != nil {
		log.Warnf("Error creating docker client from environment: %v", err)

		// Last try to create it without the environment option
		cli, err = newDockerClient()
		if err != nil {
			return nil, errors.Errorf("Cannot create docker client: %v", err)
		}
	}

	cli.NegotiateAPIVersion(ctx)
	return cli, nil
}

func newDockerClient() (*Client, error) {
	cli, err := dockerclient.NewClientWithOpts()
	if err != nil {
		return nil, errors.Errorf("Couldn't create docker client: %s", err)
	}

	return &Client{
		CommonAPIClient: cli,
	}, nil
}

func newDockerClientFromEnvironment() (*Client, error) {
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv)
	if err != nil {
		return nil, errors.Errorf("Couldn't create docker client: %s", err)
	}

	return &Client{
		CommonAPIClient: cli,
	}, nil
}

func (c *Client) RawExec(ctx context.Context, container string, user string, cmd []string, stdin io.ReadCloser, stdout io.Writer, stderr io.Writer) error {
	execConfig := types.ExecConfig{
		User:         user,
		AttachStdin:  false,
		AttachStderr: false,
		AttachStdout: false,
		Cmd:          cmd,
	}
	if stdin != nil {
		execConfig.AttachStdin = true
	}
	if stderr != nil {
		execConfig.AttachStderr = true
	}
	if stdout != nil {
		execConfig.AttachStdout = true
	}

	resp, execID, err := c.Exec(ctx, container, execConfig)
	if err != nil {
		return err
	}
	defer resp.Close()

	streamer := &HijackedIOStreamer{
		InputStream:  stdin,
		OutputStream: stdout,
		ErrorStream:  stderr,
		Resp:         resp,
	}

	err = streamer.Stream(ctx)
	if err != nil {
		return err
	}

	return c.getExecExitStatus(ctx, execID)
}

func (c *Client) getExecExitStatus(ctx context.Context, execID string) error {
	resp, err := c.ContainerExecInspect(ctx, execID)
	if err != nil {
		// If we can't connect, then the daemon probably died.
		if !dockerclient.IsErrConnectionFailed(err) {
			return err
		}
		return cli.StatusError{StatusCode: -1}
	}
	status := resp.ExitCode
	if status != 0 {
		return cli.StatusError{StatusCode: status}
	}
	return nil
}

func (c *Client) Exec(ctx context.Context, container string, execConfig types.ExecConfig) (types.HijackedResponse, string, error) {
	resp, err := c.ContainerExecCreate(ctx, container, execConfig)
	if err != nil {
		return types.HijackedResponse{}, "", err
	}

	hijackedResp, err := c.ContainerExecAttach(ctx, resp.ID, types.ExecStartCheck{})
	if err != nil {
		return types.HijackedResponse{}, "", err
	}

	return hijackedResp, resp.ID, nil
}

// ParseProxyConfig parses the proxy config from the ~/.docker/config.json
func (c *Client) ParseProxyConfig(buildArgs map[string]*string) map[string]*string {
	dockerConfig, err := LoadDockerConfig()
	if err == nil {
		buildArgs = dockerConfig.ParseProxyConfig(c.DaemonHost(), buildArgs)
	}

	return buildArgs
}
