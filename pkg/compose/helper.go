package compose

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	"github.com/blang/semver"
	composecli "github.com/compose-spec/compose-go/v2/cli"
	composetypes "github.com/compose-spec/compose-go/v2/types"
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

func LoadDockerComposeProject(ctx context.Context, paths []string, envFiles []string) (*composetypes.Project, error) {
	projectOptions, err := composecli.NewProjectOptions(
		paths,
		composecli.WithOsEnv,
		composecli.WithEnvFiles(envFiles...),
		composecli.WithDotEnv,
		composecli.WithDefaultProfiles(),
	)
	if err != nil {
		return nil, err
	}

	project, err := composecli.ProjectFromOptions(ctx, projectOptions)
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

func (h *ComposeHelper) FindDevContainer(ctx context.Context, projectName, serviceName string) (*config.ContainerDetails, error) {
	containerIDs, err := h.Docker.FindContainer(ctx, []string{
		fmt.Sprintf("%s=%s", ProjectLabel, projectName),
		fmt.Sprintf("%s=%s", ServiceLabel, serviceName),
	})
	if err != nil {
		return nil, err
	} else if len(containerIDs) == 0 {
		return nil, nil
	}

	containerDetails, err := h.Docker.InspectContainers(ctx, containerIDs)
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

func (h *ComposeHelper) Stop(ctx context.Context, projectName string, args []string) error {
	buildArgs := []string{"--project-name", projectName}
	buildArgs = append(buildArgs, args...)
	buildArgs = append(buildArgs, "stop")

	out, err := h.buildCmd(ctx, buildArgs...).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", string(out))
	}

	return nil
}

func (h *ComposeHelper) Remove(ctx context.Context, projectName string, args []string) error {
	buildArgs := []string{"--project-name", projectName}
	buildArgs = append(buildArgs, args...)
	buildArgs = append(buildArgs, "down")

	out, err := h.buildCmd(ctx, buildArgs...).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s", string(out))
	}

	return nil
}

func (h *ComposeHelper) GetDefaultImage(projectName, serviceName string) (string, error) {
	version, err := semver.Parse(strings.TrimPrefix(h.Version, "v"))
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

func (h *ComposeHelper) FindProjectFiles(ctx context.Context, projectName string) ([]string, error) {
	buildArgs := []string{"--project-name", projectName}
	buildArgs = append(buildArgs, "ls", "-a", "--filter", "name="+projectName, "--format", "json")

	rawOut, err := h.buildCmd(ctx, buildArgs...).CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "%s", string(rawOut))
	}

	type composeOutput struct {
		Name        string
		Status      string
		ConfigFiles string
	}
	var composeOutputs []composeOutput
	if err := json.Unmarshal(rawOut, &composeOutputs); err != nil {
		return nil, errors.Wrapf(err, "parse compose output")
	}

	// no compose project found
	if len(composeOutputs) == 0 {
		return nil, nil
	}

	// Parse project files of first match
	projectFiles := strings.Split(composeOutputs[0].ConfigFiles, ",")
	return projectFiles, nil
}

func (h *ComposeHelper) GetProjectName(runnerID string) string {
	return h.toProjectName(runnerID)
}

func (h *ComposeHelper) toProjectName(projectName string) string {
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
