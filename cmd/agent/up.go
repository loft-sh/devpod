package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/devcontainer"
	"github.com/loft-sh/devpod/pkg/extract"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/scripts"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path/filepath"
)

// UpCmd holds the up cmd flags
type UpCmd struct {
	flags.GlobalFlags
}

// NewUpCmd creates a new ssh command
func NewUpCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &UpCmd{
		GlobalFlags: *flags,
	}
	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Starts a new devcontainer",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.Background())
		},
	}
	return upCmd
}

// Run runs the command logic
func (cmd *UpCmd) Run(ctx context.Context) error {
	// create a grpc client
	tunnelClient, err := agent.NewTunnelClient(os.Stdin, os.Stdout, true)
	if err != nil {
		return errors.Wrap(err, "create tunnel client")
	}

	// create debug logger
	logger := agent.NewTunnelLogger(ctx, tunnelClient, cmd.Debug)

	// get workspace
	workspace, err := getWorkspace(ctx, tunnelClient)
	if err != nil {
		return err
	}

	// install dependencies
	err = InstallDependencies(logger)
	if err != nil {
		return err
	}

	// git clone repository
	workspaceDir, err := cmd.prepareWorkspace(ctx, workspace, tunnelClient, logger)
	if err != nil {
		return err
	}

	// create devcontainer
	err = DevContainerUp(workspace.ID, workspaceDir, logger)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *UpCmd) prepareWorkspace(ctx context.Context, workspace *provider2.Workspace, client tunnel.TunnelClient, log log.Logger) (string, error) {
	workspaceDir, err := getWorkspaceDir(workspace.ID)
	if err != nil {
		return "", err
	}

	// check if it already exists
	_, err = os.Stat(workspaceDir)
	if err == nil {
		return workspaceDir, nil
	}

	// create workspace folder
	err = os.MkdirAll(workspaceDir, 0755)
	if err != nil {
		return "", err
	}

	// check what type of workspace this is
	if workspace.Source.GitRepository != "" {
		return CloneRepository(workspaceDir, workspace.Source.GitRepository, log)
	} else if workspace.Source.LocalFolder != "" {
		return DownloadLocalFolder(ctx, workspaceDir, client, log)
	} else if workspace.Source.Image != "" {
		return PrepareImage(workspaceDir, workspace.Source.Image)
	}

	return "", fmt.Errorf("either workspace repository, image or local-folder is required")
}

func getWorkspace(ctx context.Context, client tunnel.TunnelClient) (*provider2.Workspace, error) {
	workspaceResult, err := client.Workspace(ctx, &tunnel.Empty{})
	if err != nil {
		return nil, err
	}

	workspace := &provider2.Workspace{}
	err = json.Unmarshal([]byte(workspaceResult.Workspace), workspace)
	if err != nil {
		return nil, errors.Wrap(err, "parse workspace")
	}

	return workspace, nil
}

func DownloadLocalFolder(ctx context.Context, workspaceDir string, client tunnel.TunnelClient, log log.Logger) (string, error) {
	log.Infof("Upload folder to server")
	stream, err := client.ReadWorkspace(ctx, &tunnel.Empty{})
	if err != nil {
		return "", errors.Wrap(err, "read workspace")
	}

	err = extract.Extract(agent.NewStreamReader(stream), workspaceDir)
	if err != nil {
		return "", errors.Wrap(err, "extract local folder")
	}

	return workspaceDir, nil
}

func PrepareImage(workspaceDir, image string) (string, error) {
	// create a .devcontainer.json with the image
	err := os.WriteFile(filepath.Join(workspaceDir, ".devcontainer.json"), []byte(`{
  "image": "`+image+`"
}`), 0666)
	if err != nil {
		return "", err
	}

	return workspaceDir, nil
}

func DevContainerUp(id, workspaceFolder string, log log.Logger) error {
	err := devcontainer.NewRunner(agent.RemoteDevPodHelperLocation, agent.DefaultAgentDownloadURL, workspaceFolder, id, log).Up()
	if err != nil {
		return err
	}

	return nil
}

func CloneRepository(workspaceDir, repository string, log log.Logger) (string, error) {
	// run git command
	writer := log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	gitCommand := exec.Command("git", "clone", repository, workspaceDir)
	gitCommand.Stdout = writer
	gitCommand.Stderr = writer
	err := gitCommand.Run()
	if err != nil {
		return "", errors.Wrap(err, "error cloning repository")
	}

	return workspaceDir, nil
}

func InstallDependencies(log log.Logger) error {
	if !command.Exists("docker") {
		writer := log.Writer(logrus.InfoLevel, false)
		defer writer.Close()

		shellCommand := exec.Command("sh", "-c", scripts.InstallDocker)
		shellCommand.Stdout = writer
		shellCommand.Stderr = writer
		err := shellCommand.Run()
		if err != nil {
			return err
		}
	}

	return nil
}

func getWorkspaceDir(id string) (string, error) {
	// workspace folder
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	workspaceDir := filepath.Join(homeDir, ".devpod", "workspace", id)
	return workspaceDir, nil
}
