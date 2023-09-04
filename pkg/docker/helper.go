package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/loft-sh/log/scanner"
	perrors "github.com/pkg/errors"
)

type DockerHelper struct {
	DockerCommand string
	// allow command to have a custom environment
	Environment []string
}

func (r *DockerHelper) GPUSupportEnabled() (bool, error) {
	out, err := r.buildCmd(context.TODO(), "info", "-f", "{{.Runtimes.nvidia}}").Output()
	if err != nil {
		return false, command.WrapCommandError(out, err)
	}

	return strings.Contains(string(out), "nvidia-container-runtime"), nil
}

func (r *DockerHelper) FindDevContainer(ctx context.Context, labels []string) (*config.ContainerDetails, error) {
	containers, err := r.FindContainer(ctx, labels)
	if err != nil {
		return nil, fmt.Errorf("docker ps: %w", err)
	} else if len(containers) == 0 {
		return nil, nil
	}

	return r.FindContainerByID(ctx, containers)
}

func (r *DockerHelper) FindContainerByID(ctx context.Context, containerIds []string) (*config.ContainerDetails, error) {
	containerDetails, err := r.InspectContainers(ctx, containerIds)
	if err != nil {
		return nil, err
	}

	// find matching container
	for _, details := range containerDetails {
		if strings.ToLower(details.State.Status) != "removing" {
			details.State.Status = strings.ToLower(details.State.Status)
			return &details, nil
		}
	}

	return nil, nil
}

func (r *DockerHelper) DeleteVolume(ctx context.Context, volume string) error {
	if volume == "" {
		return nil
	}

	// If volume does not exist, just exit
	out, err := r.buildCmd(ctx, "volume", "list", "-q", "--filter", "name="+volume).CombinedOutput()
	if err != nil {
		return nil
	}
	if len(out) == 0 {
		return nil
	}

	out, err = r.buildCmd(ctx, "volume", "rm", volume).CombinedOutput()
	if err != nil {
		return perrors.Wrapf(err, "%s", string(out))
	}

	return nil
}

func (r *DockerHelper) Stop(ctx context.Context, id string) error {
	out, err := r.buildCmd(ctx, "stop", id).CombinedOutput()
	if err != nil {
		return perrors.Wrapf(err, "%s", string(out))
	}

	return nil
}

func (r *DockerHelper) Remove(ctx context.Context, id string) error {
	out, err := r.buildCmd(ctx, "rm", id).CombinedOutput()
	if err != nil {
		return perrors.Wrapf(err, "%s", string(out))
	}

	return nil
}

func (r *DockerHelper) Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	return r.RunWithDir(ctx, "", args, stdin, stdout, stderr)
}

func (r *DockerHelper) RunWithDir(ctx context.Context, dir string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	cmd := r.buildCmd(ctx, args...)
	cmd.Dir = dir
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func (r *DockerHelper) StartContainer(ctx context.Context, containerId string) error {
	out, err := r.buildCmd(ctx, "start", containerId).CombinedOutput()
	if err != nil {
		return perrors.Wrapf(err, "start command: %v", string(out))
	}

	container, err := r.FindContainerByID(ctx, []string{containerId})
	if err != nil {
		return err
	} else if container == nil {
		return fmt.Errorf("container not found")
	}

	return nil
}

func (r *DockerHelper) InspectImage(ctx context.Context, imageName string, tryRemote bool) (*config.ImageDetails, error) {
	imageDetails := []*config.ImageDetails{}
	err := r.Inspect(ctx, []string{imageName}, "image", &imageDetails)
	if err != nil {
		// try remote?
		if !tryRemote {
			return nil, err
		}

		imageConfig, _, err := image.GetImageConfig(imageName)
		if err != nil {
			return nil, perrors.Wrap(err, "get image config remotely")
		}

		return &config.ImageDetails{
			ID: imageName,
			Config: config.ImageDetailsConfig{
				User:       imageConfig.Config.User,
				Env:        imageConfig.Config.Env,
				Labels:     imageConfig.Config.Labels,
				Entrypoint: imageConfig.Config.Entrypoint,
				Cmd:        imageConfig.Config.Cmd,
			},
		}, nil
	} else if len(imageDetails) == 0 {
		return nil, fmt.Errorf("cannot find image details for %s", imageName)
	}

	return imageDetails[0], nil
}

func (r *DockerHelper) InspectContainers(ctx context.Context, ids []string) ([]config.ContainerDetails, error) {
	details := []config.ContainerDetails{}
	err := r.Inspect(ctx, ids, "container", &details)
	if err != nil {
		return nil, err
	}

	return details, nil
}

func (r *DockerHelper) IsPodman() bool {
	args := []string{"--version"}

	out, err := r.buildCmd(context.TODO(), args...).Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(out), "podman")
}

func (r *DockerHelper) Inspect(ctx context.Context, ids []string, inspectType string, obj interface{}) error {
	args := []string{"inspect", "--type", inspectType}
	args = append(args, ids...)
	out, err := r.buildCmd(ctx, args...).Output()
	if err != nil {
		return fmt.Errorf("inspect container: %w", command.WrapCommandError(out, err))
	}

	err = json.Unmarshal(out, obj)
	if err != nil {
		return perrors.Wrap(err, "parse inspect output")
	}

	return nil
}

// FindContainer will try to find a container based on the input labels.
// If no container is found, it will search for the labels manually inspecting
// containers.
func (r *DockerHelper) FindContainer(ctx context.Context, labels []string) ([]string, error) {
	args := []string{"ps", "-q", "-a"}
	for _, label := range labels {
		args = append(args, "--filter", "label="+label)
	}

	out, err := r.buildCmd(ctx, args...).Output()
	if err != nil {
		// fallback to manual search
		return r.FindContainerJSON(ctx, labels)
	}

	arr := []string{}
	scan := scanner.NewScanner(bytes.NewReader(out))
	for scan.Scan() {
		arr = append(arr, strings.TrimSpace(scan.Text()))
	}

	return arr, nil
}

// FindContainerJSON will manually search for containers with matching labels.
// This is useful in case the `--filter` doesn't work.
func (r *DockerHelper) FindContainerJSON(ctx context.Context, labels []string) ([]string, error) {
	args := []string{"ps", "-q", "-a"}
	out, err := r.buildCmd(ctx, args...).Output()
	if err != nil {
		return nil, command.WrapCommandError(out, err)
	}

	result := []string{}

	ids := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
	for _, id := range ids {
		id = strings.TrimSpace(id)
		found := true

		containers, err := r.InspectContainers(ctx, []string{id})
		if err != nil {
			continue
		}

		for _, label := range labels {
			key := strings.Split(label, "=")[0]
			value := strings.Join(strings.Split(label, "=")[1:], "=")

			found = containers[0].Config.Labels[key] == value
		}

		if found {
			result = append(result, id)
		}
	}

	return result, nil
}

func (r *DockerHelper) buildCmd(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, r.DockerCommand, args...)
	if r.Environment != nil {
		cmd.Env = append(os.Environ(), r.Environment...)
	}
	return cmd
}
