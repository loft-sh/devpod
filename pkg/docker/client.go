package docker

import (
	"context"

	dockerclient "github.com/docker/docker/client"
	"github.com/loft-sh/log"
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
