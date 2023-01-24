package agent

import (
	"fmt"
	"github.com/loft-sh/devpod/scripts"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	DockerIDLabel = "dev.containers.id"
)

// UpCmd holds the up cmd flags
type UpCmd struct {
	ID         string
	Repository string
}

// NewUpCmd creates a new ssh command
func NewUpCmd() *cobra.Command {
	cmd := &UpCmd{}
	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Starts a new devcontainer",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}

	upCmd.Flags().StringVar(&cmd.ID, "id", "", "The id of the dev container")
	upCmd.Flags().StringVar(&cmd.Repository, "repository", "", "The repository to clone and create")
	return upCmd
}

// Run runs the command logic
func (cmd *UpCmd) Run(_ *cobra.Command, _ []string) error {
	err := InstallDependencies()
	if err != nil {
		return err
	}

	// git clone repository
	workspaceDir, err := CloneRepository(cmd.ID, cmd.Repository)
	if err != nil {
		return err
	}

	// run devcontainer up
	err = DevContainerUp(cmd.ID, workspaceDir)
	if err != nil {
		return err
	}

	return nil
}

func DevContainerUp(id, workspaceFolder string) error {
	devContainerCommand := fmt.Sprintf("devcontainer up  --workspace-folder %s --id-label %s=%s", workspaceFolder, DockerIDLabel, id)
	if os.Getuid() != 0 {
		devContainerCommand = "sudo " + devContainerCommand
	}

	execCommand := exec.Command("sh", "-c", devContainerCommand)
	execCommand.Stdout = os.Stdout
	execCommand.Stderr = os.Stderr
	err := execCommand.Run()
	if err != nil {
		return err
	}

	return nil
}

func InstallDependencies() error {
	if !commandExists("docker") {
		shellCommand := exec.Command("sh", "-c", scripts.InstallDocker)
		shellCommand.Stdout = os.Stdout
		shellCommand.Stderr = os.Stderr
		err := shellCommand.Run()
		if err != nil {
			return err
		}
	}

	if !commandExists("devcontainer") {
		shellCommand := exec.Command("sh", "-c", scripts.InstallDevContainer)
		shellCommand.Stdout = os.Stdout
		shellCommand.Stderr = os.Stderr
		err := shellCommand.Run()
		if err != nil {
			return err
		}
	}

	return nil
}

func CloneRepository(id, repository string) (string, error) {
	workspaceDir, err := getWorkspaceDir(id)
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

	// clone the repository
	if repository == "-" {
		// TODO: extract tar archive
		return "", nil
	}

	// run git command
	gitCommand := exec.Command("git", "clone", repository, workspaceDir)
	gitCommand.Stdout = os.Stdout
	gitCommand.Stderr = os.Stderr
	err = gitCommand.Run()
	if err != nil {
		return "", errors.Wrap(err, "error cloning repository")
	}

	return workspaceDir, nil
}

func getWorkspaceDir(id string) (string, error) {
	// workspace folder
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	workspaceDir := filepath.Join(homeDir, "devpod", "workspace", id)
	return workspaceDir, nil
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
