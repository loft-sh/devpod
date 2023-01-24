package devcontainer

import (
	"bytes"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/scanner"
	"os/exec"
	"strings"
)

func (r *Runner) runSingleContainer(config *config.DevContainerConfig) error {
	return nil
}

func (r *Runner) InspectContainer(id string) error {
	return nil
}

func (r *Runner) FindContainer(labels []string) ([]string, error) {
	args := []string{r.DockerCommand, "ps", "-q", "-a"}
	for _, label := range labels {
		args = append(args, "--filter", "label="+label)
	}

	out, err := exec.Command(args[0], args[1:]...).Output()
	if err != nil {
		return nil, err
	}

	arr := []string{}
	scan := scanner.NewScanner(bytes.NewReader(out))
	for scan.Scan() {
		arr = append(arr, strings.TrimSpace(scan.Text()))
	}

	return arr, nil
}
