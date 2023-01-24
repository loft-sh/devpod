package cmd

import (
	"bytes"
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider/gcp"
	"github.com/loft-sh/devpod/pkg/provider/types"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/template"
	"github.com/loft-sh/devpod/pkg/token"
	"github.com/loft-sh/devpod/scripts"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
)

// SSHCmd holds the ssh cmd flags
type SSHCmd struct {
	Stdio         bool
	JumpContainer bool

	Configure bool
	ID        string

	ShowAgentCommand bool
}

// NewSSHCmd creates a new ssh command
func NewSSHCmd() *cobra.Command {
	cmd := &SSHCmd{}
	sshCmd := &cobra.Command{
		Use:   "ssh",
		Short: "Starts a new ssh session to a workspace",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}

	sshCmd.Flags().StringVar(&cmd.ID, "id", "", "The id of the workspace to use")
	sshCmd.Flags().BoolVar(&cmd.Configure, "configure", false, "If true will configure ssh for the given workspace")
	sshCmd.Flags().BoolVar(&cmd.Stdio, "stdio", false, "If true will tunnel connection through stdout and stdin")
	sshCmd.Flags().BoolVar(&cmd.JumpContainer, "jump-container", false, "If true will jump into the container")
	sshCmd.Flags().BoolVar(&cmd.ShowAgentCommand, "show-agent-command", false, "If true will show with which flags the agent needs to be executed remotely")
	return sshCmd
}

// Run runs the command logic
func (cmd *SSHCmd) Run(_ *cobra.Command, _ []string) error {
	if cmd.ShowAgentCommand {
		return cmd.showAgentCommand()
	}
	if cmd.Configure {
		return configureSSH(cmd.ID, "root")
	}

	handler, err := gcp.NewGCPProvider(log.Default).RemoteCommandHost(context.Background(), &types.Workspace{ID: "test"}, types.RemoteCommandOptions{})
	if err != nil {
		return err
	}
	defer handler.Close()

	if cmd.JumpContainer {
		return cmd.jumpContainer(handler, cmd.ID)
	}
	if cmd.Stdio {
		// TODO: implement agent call here
		return nil
	}

	// TODO: Implement regular ssh client here
	return nil
}

func (cmd *SSHCmd) jumpContainer(handler types.RemoteCommandHandler, workspace string) error {
	// install devpod
	err := installDevPod(handler)
	if err != nil {
		return errors.Wrap(err, "install devpod")
	}

	// get token
	t, err := token.GenerateToken()
	if err != nil {
		return err
	}

	// tunnel to container
	err = handler.Run(context.Background(), fmt.Sprintf("sudo %s agent container-tunnel --id %s --token %s", agent.RemoteDevPodHelperLocation, workspace, t), os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		return err
	}

	return nil
}

func installDevPod(handler types.RemoteCommandHandler) error {
	installed, err := isDevPodInstalled(handler)
	if err != nil {
		return err
	} else if installed {
		return nil
	}

	// install devpod into the ssh machine
	t, err := template.FillTemplate(scripts.InstallDevPodTemplate, map[string]string{
		"BaseUrl": "https://github.com/FabianKramm/foundation/releases/download/test",
	})
	if err != nil {
		return err
	}

	// install devpod
	buf := &bytes.Buffer{}
	err = handler.Run(context.TODO(), t, nil, buf, buf)
	if err != nil {
		return errors.Wrapf(err, "install devpod: %s", buf.String())
	}

	return nil
}

func isDevPodInstalled(handler types.RemoteCommandHandler) (bool, error) {
	err := handler.Run(context.Background(), fmt.Sprintf("%s version", agent.RemoteDevPodHelperLocation), nil, nil, nil)
	if err != nil {
		return false, nil
	}

	return true, nil
}

func (cmd *SSHCmd) showAgentCommand() error {
	t, _ := token.GenerateToken()
	fmt.Println(fmt.Sprintf("devpod agent ssh-server --token %s", t))
	return nil
}

func configureSSH(id, user string) error {
	err := devssh.ConfigureSSHConfig(id, user, log.Default)
	if err != nil {
		return err
	}

	return nil
}
