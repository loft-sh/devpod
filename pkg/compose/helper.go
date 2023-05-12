package compose

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/blang/semver"
	composecli "github.com/compose-spec/compose-go/cli"
	composetypes "github.com/compose-spec/compose-go/types"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/docker"
	"github.com/pkg/errors"
)

const (
	DockerCommand        = "docker"
	DockerComposeCommand = "docker-compose"
	ProjectLabel         = "com.docker.compose.project"
	ServiceLabel         = "com.docker.compose.service"
)

func LoadDockerComposeProject(paths []string, envFiles []string) (*composetypes.Project, error) {
	projectOptions, err := composecli.NewProjectOptions(paths, composecli.WithEnvFiles(envFiles...))
	if err != nil {
		return nil, err
	}

	project, err := composecli.ProjectFromOptions(projectOptions)
	if err != nil {
		return nil, err
	}

	return project, nil
}

type ComposeHelper struct {
	Command string
	Version string
	Args    []string
	Docker  *docker.DockerHelper
}

func NewComposeHelper(dockerComposeCLI string, dockerHelper *docker.DockerHelper) (*ComposeHelper, error) {
	dockerCLI := dockerHelper.DockerCommand
	if dockerCLI == "" {
		dockerCLI = DockerCommand
	}

	if dockerComposeCLI == "" {
		dockerComposeCLI = DockerComposeCommand
	}

	if out, err := exec.Command(dockerComposeCLI, "version", "--short").Output(); err == nil {
		return &ComposeHelper{
			Command: dockerComposeCLI,
			Version: strings.TrimSpace(string(out)),
			Args:    []string{},
			Docker:  dockerHelper,
		}, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	out, err := exec.Command(dockerCLI, "compose", "version", "--short").Output()
	if err == nil {
		return &ComposeHelper{
			Command: dockerCLI,
			Version: strings.TrimSpace(string(out)),
			Args:    []string{"compose"},
			Docker:  dockerHelper,
		}, nil
	}

	return nil, err
}

func (h *ComposeHelper) FindDevContainer(projectName, serviceName string) (*config.ContainerDetails, error) {
	containerIDs, err := h.Docker.FindContainer([]string{
		fmt.Sprintf("%s=%s", ProjectLabel, projectName),
		fmt.Sprintf("%s=%s", ServiceLabel, serviceName),
	})
	if err != nil {
		return nil, err
	} else if len(containerIDs) == 0 {
		return nil, nil
	}

	containerDetails, err := h.Docker.InspectContainers(containerIDs)
	if err != nil {
		return nil, err
	}

	for _, details := range containerDetails {
		if details.State.Status != "removing" {
			return &details, nil
		}
	}

	return nil, nil
}

func (h *ComposeHelper) Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	cmd := h.buildCmd(ctx, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func (h *ComposeHelper) Stop(projectName string) error {
	out, err := h.buildCmd(context.TODO(), "--project-name", projectName, "stop").CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", string(out))
	}

	return nil
}

func (h *ComposeHelper) Remove(projectName string) error {
	out, err := h.buildCmd(context.TODO(), "--project-name", projectName, "down").CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", string(out))
	}

	return nil
}

func (h *ComposeHelper) GetDefaultImage(projectName, serviceName string) (string, error) {
	version, err := semver.Parse(h.Version)
	if err != nil {
		return "", err
	}

	earlierVersion, err := semver.New("2.8.0")
	if err != nil {
		return "", err
	}

	if version.Compare(*earlierVersion) == -1 {
		return fmt.Sprintf("%s_%s", projectName, serviceName), nil
	}

	return fmt.Sprintf("%s-%s", projectName, serviceName), nil
}

func (h *ComposeHelper) ToProjectName(projectName string) string {
	useNewProjectNameFormat, _ := h.useNewProjectName()
	if !useNewProjectNameFormat {
		return regexp.MustCompile("[^a-z0-9]").ReplaceAllString(strings.ToLower(projectName), "")
	}

	return regexp.MustCompile("[^-_a-z0-9]").ReplaceAllString(strings.ToLower(projectName), "")
}

func (h *ComposeHelper) buildCmd(ctx context.Context, args ...string) *exec.Cmd {
	var allArgs []string
	allArgs = append(allArgs, h.Args...)
	allArgs = append(allArgs, args...)
	return exec.CommandContext(ctx, h.Command, allArgs...)
}

func (h *ComposeHelper) useNewProjectName() (bool, error) {
	version, err := semver.Parse(h.Version)
	if err != nil {
		return false, err
	}

	earlierVersion, err := semver.New("1.12.0")
	if err != nil {
		return false, err
	}

	if version.Compare(*earlierVersion) == -1 {
		return false, nil
	}

	return true, nil
}
