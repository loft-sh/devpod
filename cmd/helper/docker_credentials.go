package helper

import (
	"fmt"
	"os/user"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/credentials"
	"github.com/loft-sh/devpod/pkg/dockercredentials"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

type DockerCredentialsHelperCmd struct {
	*flags.GlobalFlags

	WorkspaceID string
}

func NewDockerCredentialsHelperCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &DockerCredentialsHelperCmd{
		GlobalFlags: flags,
	}
	fleetCmd := &cobra.Command{
		Use:   "setup-docker-credentials-helper",
		Short: "Setup the docker credentials helper manually",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}

	return fleetCmd
}

func (c *DockerCredentialsHelperCmd) Run(cmd *cobra.Command, _ []string) error {
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("get current user: %w", err)
	}

	port, err := credentials.GetPort()
	if err != nil {
		return fmt.Errorf("get port: %w", err)
	}

	return dockercredentials.ConfigureCredentialsContainer(u.Name, port, log.Default)
}
