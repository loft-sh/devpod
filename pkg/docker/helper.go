package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/loft-sh/devpod/pkg/scanner"
	"github.com/pkg/errors"
	"io"
	"os/exec"
	"strings"
)

type ImageDetails struct {
	Id     string
	Config ImageDetailsConfig
}

type ImageDetailsConfig struct {
	User       string
	Env        []string
	Labels     map[string]string
	Entrypoint []string
	Cmd        []string
}

type ContainerDetails struct {
	Id              string
	Created         string
	Name            string
	State           ContainerDetailsState
	Config          ContainerDetailsConfig
	Mounts          []ContainerDetailsMount
	NetworkSettings ContainerDetailsNetworkSettings
	Ports           []ContainerDetailsPort
}

type ContainerDetailsPort struct {
	IP          string
	PrivatePort int
	PublicPort  int
	Type        string
}

type ContainerDetailsNetworkSettings struct {
	Ports map[string][]ContainerDetailsNetworkSettingsPort
}

type ContainerDetailsNetworkSettingsPort struct {
	HostIp   string
	HostPort string
}

type ContainerDetailsMount struct {
	Type        string
	Name        string
	Source      string
	Destination string
}

type ContainerDetailsConfig struct {
	Image  string
	User   string
	Env    []string
	Labels map[string]string
}

type ContainerDetailsState struct {
	Status     string
	StartedAt  string
	FinishedAt string
}

type DockerHelper struct {
	DockerCommand string
}

func ContainerToImageDetails(containerDetails *ContainerDetails) *ImageDetails {
	return &ImageDetails{
		Id: containerDetails.Id,
		Config: ImageDetailsConfig{
			User:   containerDetails.Config.User,
			Env:    containerDetails.Config.Env,
			Labels: containerDetails.Config.Labels,
		},
	}
}

func (r *DockerHelper) GPUSupportEnabled() (bool, error) {
	out, err := r.buildCmd("info", "-f", "{{.Runtimes.nvidia}}").Output()
	if err != nil {
		return false, err
	}

	return strings.Contains(string(out), "nvidia-container-runtime"), nil
}

func (r *DockerHelper) FindDevContainer(labels []string) (*ContainerDetails, error) {
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

func (r *DockerHelper) InspectImage(imageName string, tryRemote bool) (*ImageDetails, error) {
	imageDetails := []*ImageDetails{}
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

		return &ImageDetails{
			Id: imageName,
			Config: ImageDetailsConfig{
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

func (r *DockerHelper) InspectContainers(ids []string) ([]ContainerDetails, error) {
	details := []ContainerDetails{}
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

func (r *DockerHelper) Tunnel(agentPath, agentDownloadURL string, containerID string, token string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	// inject agent
	err := agent.InjectAgent(func(command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
		args := []string{"exec", "-i", "-u", "root", containerID, "sh", "-c", command}
		return r.Run(args, stdin, stdout, stderr)
	}, agentPath, agentDownloadURL, false)
	if err != nil {
		return err
	}

	// create tunnel
	args := []string{
		"exec",
		"-i",
		"-u", "root",
		containerID,
		"sh", "-c", fmt.Sprintf("%s agent ssh-server --token %s --stdio", agent.RemoteDevPodHelperLocation, token),
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
			return nil, errors.Wrapf(err, "find container: %s%s", string(exitError.Stderr), string(out))
		}

		return nil, errors.Wrapf(err, "find container: %s", string(out))
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
