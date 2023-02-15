package devcontainer

import (
	"fmt"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

func (r *Runner) startDevContainer(parsedConfig *config.DevContainerConfig, mergedConfig *config.MergedDevContainerConfig, imageName, workspaceMount string, labels []string, imageDetails *config.ImageDetails) error {
	args := []string{
		"run",
		"--sig-proxy=false",
	}

	// add ports
	for _, appPort := range parsedConfig.AppPort {
		intPort, err := strconv.Atoi(appPort)
		if err != nil {
			args = append(args, "-p", appPort)
		} else {
			args = append(args, "-p", fmt.Sprintf("127.0.0.1:%d:%d", intPort, intPort))
		}
	}

	// workspace mount
	if workspaceMount != "" {
		args = append(args, "--mount", workspaceMount)
	}

	// override container user
	if mergedConfig.ContainerUser != "" {
		args = append(args, "-u", mergedConfig.ContainerUser)
	}

	// container env
	for k, v := range mergedConfig.ContainerEnv {
		args = append(args, "-e", k+"="+v)
	}

	// security options
	if mergedConfig.Init != nil && *mergedConfig.Init {
		args = append(args, "--init")
	}
	if mergedConfig.Privileged != nil && *mergedConfig.Privileged {
		args = append(args, "--privileged")
	}
	for _, capAdd := range mergedConfig.CapAdd {
		args = append(args, "--cap-add", capAdd)
	}
	for _, securityOpt := range mergedConfig.SecurityOpt {
		args = append(args, "--security-opt", securityOpt)
	}

	// mounts
	for _, mount := range mergedConfig.Mounts {
		args = append(args, "--mount", mount.String())
	}

	// labels
	for _, label := range labels {
		args = append(args, "-l", label)
	}

	// check GPU
	if parsedConfig.HostRequirements != nil && parsedConfig.HostRequirements.GPU {
		enabled, _ := r.Docker.GPUSupportEnabled()
		if enabled {
			args = append(args, "--gpus", "all")
		}
	}

	// run detached
	args = append(args, "-d")

	// add entrypoint
	args = append(args, "--entrypoint", "/bin/sh")

	// image name
	args = append(args, imageName)

	// entrypoint
	customEntrypoints := mergedConfig.Entrypoints
	cmd := []string{"-c", `echo Container started
trap "exit 0" 15
` + strings.Join(customEntrypoints, "\n") + `
exec "$@"
while sleep 1 & wait $!; do :; done`, "-"} // `wait $!` allows for the `trap` to run (synchronous `sleep` would not).
	if mergedConfig.OverrideCommand != nil && *mergedConfig.OverrideCommand == false {
		cmd = append(cmd, imageDetails.Config.Entrypoint...)
		cmd = append(cmd, imageDetails.Config.Cmd...)
	}
	args = append(args, cmd...)

	// run the command
	r.Log.Debugf("Running docker command: docker %s", strings.Join(args, " "))
	writer := r.Log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	err := r.Docker.Run(args, nil, writer, writer)
	if err != nil {
		return err
	}

	return nil
}
