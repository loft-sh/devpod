package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/loft-sh/devpod/pkg/compose"
	config2 "github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/docker"
	"github.com/loft-sh/devpod/pkg/driver"
	"github.com/loft-sh/devpod/pkg/ide/jetbrains"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func makeEnvironment(env map[string]string, log log.Logger) []string {
	if env == nil {
		return nil
	}

	ret := config.ObjectToList(env)
	if len(env) > 0 {
		log.Debugf("Use docker environment variables: %v", ret)
	}

	return ret
}

func NewDockerDriver(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) driver.DockerDriver {
	dockerCommand := "docker"
	if workspaceInfo.Agent.Docker.Path != "" {
		dockerCommand = workspaceInfo.Agent.Docker.Path
	}

	log.Debugf("Using docker command '%s'", dockerCommand)
	return &dockerDriver{
		Docker: &docker.DockerHelper{
			DockerCommand: dockerCommand,
			Environment:   makeEnvironment(workspaceInfo.Agent.Docker.Env, log),
			ContainerID:   workspaceInfo.Workspace.Source.Container,
		},
		Log: log,
	}
}

type dockerDriver struct {
	Docker  *docker.DockerHelper
	Compose *compose.ComposeHelper

	Log log.Logger
}

func (d *dockerDriver) TargetArchitecture(ctx context.Context, workspaceId string) (string, error) {
	return runtime.GOARCH, nil
}

func (d *dockerDriver) CommandDevContainer(ctx context.Context, workspaceId, user, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	container, err := d.FindDevContainer(ctx, workspaceId)
	if err != nil {
		return err
	} else if container == nil {
		return fmt.Errorf("container not found")
	}

	args := []string{"exec"}
	if stdin != nil {
		args = append(args, "-i")
	}
	args = append(args, "-u", user, container.ID, "sh", "-c", command)
	return d.Docker.Run(ctx, args, stdin, stdout, stderr)
}

func (d *dockerDriver) PushDevContainer(ctx context.Context, image string) error {
	// push image
	writer := d.Log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	// build args
	args := []string{
		"push",
		image,
	}

	// run command
	d.Log.Debugf("Running docker command: %s %s", d.Docker.DockerCommand, strings.Join(args, " "))
	err := d.Docker.Run(ctx, args, nil, writer, writer)
	if err != nil {
		return errors.Wrap(err, "push image")
	}

	return nil
}

func (d *dockerDriver) DeleteDevContainer(ctx context.Context, workspaceId string) error {
	container, err := d.FindDevContainer(ctx, workspaceId)
	if err != nil {
		return err
	} else if container == nil {
		return nil
	}

	err = d.Docker.Remove(ctx, container.ID)
	if err != nil {
		return err
	}

	return nil
}

func (d *dockerDriver) StartDevContainer(ctx context.Context, workspaceId string) error {
	container, err := d.FindDevContainer(ctx, workspaceId)
	if err != nil {
		return err
	} else if container == nil {
		return fmt.Errorf("container not found")
	}

	return d.Docker.StartContainer(ctx, container.ID)
}

func (d *dockerDriver) StopDevContainer(ctx context.Context, workspaceId string) error {
	container, err := d.FindDevContainer(ctx, workspaceId)
	if err != nil {
		return err
	} else if container == nil {
		return fmt.Errorf("container not found")
	}

	return d.Docker.Stop(ctx, container.ID)
}

func (d *dockerDriver) InspectImage(ctx context.Context, imageName string) (*config.ImageDetails, error) {
	return d.Docker.InspectImage(ctx, imageName, true)
}

func (d *dockerDriver) ComposeHelper() (*compose.ComposeHelper, error) {
	if d.Compose != nil {
		return d.Compose, nil
	}

	var err error
	d.Compose, err = compose.NewComposeHelper(compose.DockerComposeCommand, d.Docker)
	return d.Compose, err
}

func (d *dockerDriver) FindDevContainer(ctx context.Context, workspaceId string) (*config.ContainerDetails, error) {
	var containerDetails *config.ContainerDetails
	var err error
	if d.Docker.ContainerID != "" {
		containerDetails, err = d.Docker.FindContainerByID(ctx, []string{d.Docker.ContainerID})
	} else {
		containerDetails, err = d.Docker.FindDevContainer(ctx, []string{config.DockerIDLabel + "=" + workspaceId})
	}
	if err != nil {
		return nil, err
	} else if containerDetails == nil {
		return nil, nil
	}

	if containerDetails.Config.LegacyUser != "" {
		if containerDetails.Config.Labels == nil {
			containerDetails.Config.Labels = map[string]string{}
		}
		if containerDetails.Config.Labels[config.UserLabel] == "" {
			containerDetails.Config.Labels[config.UserLabel] = containerDetails.Config.LegacyUser
		}
	}

	return containerDetails, nil
}

func (d *dockerDriver) RunDevContainer(
	ctx context.Context,
	workspaceId string,
	options *driver.RunOptions,
) error {
	return fmt.Errorf("unsupported")
}

func (d *dockerDriver) RunDockerDevContainer(
	ctx context.Context,
	workspaceId string,
	options *driver.RunOptions,
	parsedConfig *config.DevContainerConfig,
	init *bool,
	ide string,
	ideOptions map[string]config2.OptionValue,
) error {
	err := d.EnsureImage(ctx, options)
	if err != nil {
		return err
	}

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
	if options.WorkspaceMount != nil {
		workspacePath := d.EnsurePath(options.WorkspaceMount)

		args = append(args, "--mount", workspacePath.String())
	}

	// override container user
	if options.User != "" {
		args = append(args, "-u", options.User)
	}

	// container env
	for k, v := range options.Env {
		args = append(args, "-e", k+"="+v)
	}

	// security options
	if init != nil && *init {
		args = append(args, "--init")
	}
	if options.Privileged != nil && *options.Privileged {
		args = append(args, "--privileged")
	}

	// In case we're using podman, let's use userns to keep
	// the ID of the user (vscode) inside the container as
	// the same of the external user.
	// This will avoid problems of mismatching chowns on the
	// project files.
	if d.Docker.IsPodman() && os.Getuid() != 0 {
		args = append(args, "--userns", "keep-id")
	}

	for _, capAdd := range options.CapAdd {
		args = append(args, "--cap-add", capAdd)
	}
	for _, securityOpt := range options.SecurityOpt {
		args = append(args, "--security-opt", securityOpt)
	}

	// mounts
	for _, mount := range options.Mounts {
		args = append(args, "--mount", mount.String())
	}

	// add ide mounts
	switch ide {
	case string(config2.IDEGoland):
		args = append(args, "--mount", jetbrains.NewGolandServer("", ideOptions, d.Log).GetVolume())
	case string(config2.IDEPyCharm):
		args = append(args, "--mount", jetbrains.NewPyCharmServer("", ideOptions, d.Log).GetVolume())
	case string(config2.IDEPhpStorm):
		args = append(args, "--mount", jetbrains.NewPhpStorm("", ideOptions, d.Log).GetVolume())
	case string(config2.IDEIntellij):
		args = append(args, "--mount", jetbrains.NewIntellij("", ideOptions, d.Log).GetVolume())
	case string(config2.IDECLion):
		args = append(args, "--mount", jetbrains.NewCLionServer("", ideOptions, d.Log).GetVolume())
	case string(config2.IDERider):
		args = append(args, "--mount", jetbrains.NewRiderServer("", ideOptions, d.Log).GetVolume())
	case string(config2.IDERubyMine):
		args = append(args, "--mount", jetbrains.NewRubyMineServer("", ideOptions, d.Log).GetVolume())
	case string(config2.IDEWebStorm):
		args = append(args, "--mount", jetbrains.NewWebStormServer("", ideOptions, d.Log).GetVolume())
	}

	// labels
	labels := append(config.GetDockerLabelForID(workspaceId), options.Labels...)
	for _, label := range labels {
		args = append(args, "-l", label)
	}

	// check GPU
	if parsedConfig.HostRequirements != nil && parsedConfig.HostRequirements.GPU == "true" {
		enabled, _ := d.Docker.GPUSupportEnabled()
		if enabled {
			args = append(args, "--gpus", "all")
		}
	}

	// runArgs
	args = append(args, parsedConfig.RunArgs...)

	// run detached
	args = append(args, "-d")

	// add entrypoint
	if options.Entrypoint != "" {
		args = append(args, "--entrypoint", options.Entrypoint)
	}

	// image name
	args = append(args, options.Image)

	// entrypoint
	args = append(args, options.Cmd...)

	// run the command
	d.Log.Debugf("Running docker command: %s %s", d.Docker.DockerCommand, strings.Join(args, " "))
	writer := d.Log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	err = d.Docker.Run(ctx, args, nil, writer, writer)
	if err != nil {
		return err
	}

	return nil
}

func (d *dockerDriver) EnsureImage(
	ctx context.Context,
	options *driver.RunOptions,
) error {
	d.Log.Infof("Inspecting image %s", options.Image)
	_, err := d.Docker.InspectImage(ctx, options.Image, false)
	if err != nil {
		d.Log.Infof("Image %s not found", options.Image)
		d.Log.Infof("Pulling image %s", options.Image)
		writer := d.Log.Writer(logrus.DebugLevel, false)
		defer writer.Close()

		return d.Docker.Pull(ctx, options.Image, nil, writer, writer)
	}
	return nil
}

func (d *dockerDriver) EnsurePath(path *config.Mount) *config.Mount {
	// in case of local windows and remote linux tcp, we need to manually do the path conversion
	if runtime.GOOS == "windows" {
		for _, v := range d.Docker.Environment {
			// we do this only is DOCKER_HOST is not docker-desktop engine, but
			// a direct TCP connection to a docker daemon running in WSL
			if strings.Contains(v, "DOCKER_HOST=tcp://") {
				unixPath := path.Source
				unixPath = strings.Replace(unixPath, "C:", "c", 1)
				unixPath = strings.ReplaceAll(unixPath, "\\", "/")
				unixPath = "/mnt/" + unixPath

				path.Source = unixPath

				return path
			}
		}
	}
	return path
}

func (d *dockerDriver) GetDevContainerLogs(ctx context.Context, workspaceId string, stdout io.Writer, stderr io.Writer) error {
	container, err := d.FindDevContainer(ctx, workspaceId)
	if err != nil {
		return err
	} else if container == nil {
		return fmt.Errorf("container not found")
	}

	return d.Docker.GetContainerLogs(ctx, container.ID, stdout, stderr)
}
