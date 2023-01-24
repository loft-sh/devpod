package devcontainer

import (
	"fmt"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/pkg/errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func NewRunner(workspaceFolder, id string, log log.Logger) *Runner {
	return &Runner{
		DockerCommand: "docker",
		
		WorkspaceFolder: workspaceFolder,
		ID:              id,
		Log:             log,
	}
}

type Runner struct {
	DockerCommand string

	WorkspaceFolder string
	ID              string
	Log             log.Logger
}

func (r *Runner) Up() error {
	parsedConfig, err := config.ParseDevContainerJSON(r.WorkspaceFolder)
	if err != nil {
		return errors.Wrap(err, "parsing devcontainer.json")
	} else if parsedConfig == nil {
		// TODO: use a default config
		return fmt.Errorf("couldn't find a devcontainer.json")
	}

	// run initializeCommand
	err = runInitializeCommand(r.WorkspaceFolder, parsedConfig, r.Log)
	if err != nil {
		return err
	}

	// check if its a compose devcontainer.json
	if isDockerFileConfig(parsedConfig) || parsedConfig.Image != "" {
		return r.runSingleContainer(parsedConfig)
	} else if len(parsedConfig.DockerComposeFile) > 0 {
		// TODO: implement
		panic("unimplemented")
	} else {
		return fmt.Errorf("dev container config is missing one of \"image\", \"dockerFile\" or \"dockerComposeFile\" properties")
	}

	// docker run --sig-proxy=false -a STDOUT -a STDERR --mount type=bind,source=/home/devpod/devpod/workspace/test,target=/workspaces/test -l dev.containers.id=test --entrypoint /bin/sh vsc-test-38392bab732ee88d03bc36cc66c16189-uid -c echo Container started
	return runDockerImage(r.WorkspaceFolder, r.ID, parsedConfig)
}

func isDockerFileConfig(config *config.DevContainerConfig) bool {
	return config.Dockerfile != "" || config.Build.Dockerfile != ""
}

func runInitializeCommand(workspaceFolder string, config *config.DevContainerConfig, log log.Logger) error {
	if len(config.InitializeCommand) == 0 {
		return nil
	}

	// should run in shell?
	var args []string
	if len(config.InitializeCommand) == 1 {
		args = []string{"sh", "-c", config.InitializeCommand[0]}
	} else {
		args = config.InitializeCommand
	}

	// run the command
	log.Infof("Running initializeCommand from devcontainer.json: '%s'", strings.Join(args, " "))
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = workspaceFolder
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func runDockerImage(workspaceFolder, id string, config *config.DevContainerConfig) error {
	args := []string{"docker", "run",
		"--mount", getWorkspaceMount(workspaceFolder, config),
		fmt.Sprintf("-l dev.containers.id=%s", id),
		"-d",
		"--entrypoint", "/bin/sh", "-c", "sleep 10000000000",
	}

	err := exec.Command(args[0], args[1:]...).Run()
	if err != nil {
		return err
	}
	return nil
}

func getWorkspaceMount(workspaceFolder string, config *config.DevContainerConfig) string {
	if config.WorkspaceMount != "" {
		return config.WorkspaceMount
	}

	containerMountFolder := "/workspaces/" + filepath.Base(workspaceFolder)
	consistency := ""
	if runtime.GOOS != "linux" {
		consistency = ",consistency='consistent'"
	}

	return fmt.Sprintf("type=bind,source=%s,target=%s%s", workspaceFolder, containerMountFolder, consistency)
}
