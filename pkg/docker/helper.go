package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/loft-sh/devpod/pkg/scanner"
	"github.com/pkg/errors"
	"io"
	"os/exec"
	"strings"
	"time"
)

type DockerHelper struct {
	DockerCommand string
}

func (r *DockerHelper) GPUSupportEnabled() (bool, error) {
	out, err := r.buildCmd("info", "-f", "{{.Runtimes.nvidia}}").Output()
	if err != nil {
		return false, err
	}

	return strings.Contains(string(out), "nvidia-container-runtime"), nil
}

func (r *DockerHelper) FindDevContainer(labels []string) (*config.ContainerDetails, error) {
	containers, err := r.FindContainer(labels)
	if err != nil {
		return nil, err
	} else if len(containers) == 0 {
		return nil, nil
	}

	containerDetails, err := r.InspectContainers(containers)
	if err != nil {
		return nil, err
	}

	// find matching container
	for _, details := range containerDetails {
		if details.State.Status != "removing" {
			return &details, nil
		}
	}

	return nil, nil
}

func (r *DockerHelper) Stop(id string) error {
	out, err := r.buildCmd("stop", id).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", string(out))
	}

	return nil
}

func (r *DockerHelper) Remove(id string) error {
	out, err := r.buildCmd("rm", id).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", string(out))
	}

	return nil
}

func (r *DockerHelper) Run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	cmd := r.buildCmd(args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func (r *DockerHelper) StartContainer(id string, labels []string) error {
	out, err := r.buildCmd("start", id).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "start command: %v", string(out))
	}

	container, err := r.FindDevContainer(labels)
	if err != nil {
		return err
	} else if container == nil {
		return fmt.Errorf("container not found")
	}

	return nil
}

func (r *DockerHelper) InspectImage(imageName string, tryRemote bool) (*config.ImageDetails, error) {
	imageDetails := []*config.ImageDetails{}
	err := r.Inspect([]string{imageName}, "image", &imageDetails)
	if err != nil {
		// try remote?
		if !tryRemote {
			return nil, err
		}

		imageConfig, _, err := image.GetImageConfig(imageName)
		if err != nil {
			return nil, err
		}

		return &config.ImageDetails{
			Id: imageName,
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

func (r *DockerHelper) InspectContainers(ids []string) ([]config.ContainerDetails, error) {
	details := []config.ContainerDetails{}
	err := r.Inspect(ids, "container", &details)
	if err != nil {
		return nil, err
	}

	return details, nil
}

func (r *DockerHelper) Inspect(ids []string, inspectType string, obj interface{}) error {
	args := []string{"inspect", "--type", inspectType}
	args = append(args, ids...)
	out, err := r.buildCmd(args...).Output()
	if err != nil {
		return errors.Wrapf(err, "inspect container: %v", string(out))
	}

	err = json.Unmarshal(out, obj)
	if err != nil {
		return errors.Wrap(err, "parse inspect output")
	}

	return nil
}

func (r *DockerHelper) Tunnel(agentPath, agentDownloadURL string, containerID string, token string, stdin io.Reader, stdout io.Writer, stderr io.Writer, trackActivity bool) error {
	// inject agent
	err := agent.InjectAgent(func(command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
		args := []string{"exec", "-i", "-u", "root", containerID, "sh", "-c", command}
		return r.Run(args, stdin, stdout, stderr)
	}, agentPath, agentDownloadURL, false, time.Second*10)
	if err != nil {
		return err
	}

	// build command
	command := fmt.Sprintf("%s helper ssh-server --token %s --stdio", agent.RemoteDevPodHelperLocation, token)
	if trackActivity {
		command += " --track-activity"
	}

	// create tunnel
	args := []string{
		"exec",
		"-i",
		"-u", "root",
		containerID,
		"sh", "-c", command,
	}
	err = r.Run(args, stdin, stdout, stderr)
	if err != nil {
		return err
	}

	return nil
}

func (r *DockerHelper) FindContainer(labels []string) ([]string, error) {
	args := []string{"ps", "-q", "-a"}
	for _, label := range labels {
		args = append(args, "--filter", "label="+label)
	}

	out, err := r.buildCmd(args...).Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("find container: %s", strings.TrimSpace(string(exitError.Stderr)))
		}

		return nil, fmt.Errorf("find container: %s", strings.TrimSpace(string(out)))
	}

	arr := []string{}
	scan := scanner.NewScanner(bytes.NewReader(out))
	for scan.Scan() {
		arr = append(arr, strings.TrimSpace(scan.Text()))
	}

	return arr, nil
}

func (r *DockerHelper) buildCmd(args ...string) *exec.Cmd {
	return exec.Command(r.DockerCommand, args...)
}
