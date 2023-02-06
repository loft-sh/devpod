package agent

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/devcontainer"
	"github.com/loft-sh/devpod/pkg/extract"
	"github.com/loft-sh/devpod/pkg/log"
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
	ID string

	Image         string
	LocalFolder   bool
	GitRepository string
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

	upCmd.Flags().StringVar(&cmd.ID, "id", "", "The id of the dev container")
	upCmd.Flags().StringVar(&cmd.Image, "image", "", "The docker image to use")
	upCmd.Flags().BoolVar(&cmd.LocalFolder, "local-folder", false, "If a local folder should be used")
	upCmd.Flags().StringVar(&cmd.GitRepository, "repository", "", "The repository to clone and create")
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

	// install dependencies
	err = InstallDependencies(logger)
	if err != nil {
		return err
	}

	// git clone repository
	workspaceDir, err := cmd.prepareWorkspace(ctx, tunnelClient, logger)
	if err != nil {
		return err
	}

	// create devcontainer
	err = DevContainerUp(cmd.ID, workspaceDir, logger)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *UpCmd) prepareWorkspace(ctx context.Context, client tunnel.TunnelClient, log log.Logger) (string, error) {
	workspaceDir, err := getWorkspaceDir(cmd.ID)
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

	if cmd.GitRepository != "" {
		return CloneRepository(workspaceDir, cmd.GitRepository, log)
	} else if cmd.LocalFolder {
		return DownloadLocalFolder(ctx, workspaceDir, client, log)
	} else if cmd.Image != "" {
		return PrepareImage(workspaceDir, cmd.Image)
	}

	return "", fmt.Errorf("either --repository, --image or --local-folder is required")
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
	err := devcontainer.NewRunner(agent.DefaultAgentDownloadURL, workspaceFolder, id, log).Up()
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
