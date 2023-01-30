package gcloud

import (
	"bytes"
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider/types"
	"github.com/pkg/errors"
	"io"
	"os/exec"
	"strings"
)

type ProviderConfig struct {
	BinaryPath string

	MachineType string
	DiskImage   string
	DiskSizeGB  int

	CreateExtraArgs []string

	Project string
	Zone    string
}

func NewProvider(config ProviderConfig, log log.Logger) (types.ServerProvider, error) {
	if config.BinaryPath == "" {
		config.BinaryPath = "gcloud"
	}

	provider := &gcloudProvider{
		Config: config,
		Log:    log,
	}

	// set defaults
	if provider.Config.Project == "" {
		defaultProject, err := provider.output(context.Background(), "config", "list", "--format", "value(core.project)")
		if err != nil {
			return nil, errors.Wrap(err, "find default project")
		}

		provider.Config.Project = strings.TrimSpace(string(defaultProject))
		if provider.Config.Project == "" {
			return nil, fmt.Errorf("please set a default project for the gcloud command")
		}
	}
	if provider.Config.Zone == "" {
		provider.Config.Zone = "europe-west1-b"
	}
	if provider.Config.MachineType == "" {
		provider.Config.MachineType = "e2-standard-2"
	}
	if provider.Config.DiskSizeGB == 0 {
		provider.Config.DiskSizeGB = 30
	}

	return provider, nil
}

type gcloudProvider struct {
	Config ProviderConfig

	Log              log.Logger
	WorkingDirectory string
}

func (g *gcloudProvider) Name() string {
	return "gcloud"
}

func (g *gcloudProvider) output(ctx context.Context, args ...string) ([]byte, error) {
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	err := g.runCommand(ctx, args, stdout, stderr, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "%s%s", stdout.String(), stderr.String())
	}

	return stdout.Bytes(), nil
}

func (g *gcloudProvider) runCommand(ctx context.Context, args []string, stdout, stderr io.Writer, stdin io.Reader) error {
	g.Log.Debugf("Run command: %s %s", g.Config.BinaryPath, strings.Join(args, " "))
	args = append(args, "--verbosity=error")
	args = append(args, "--quiet")

	cmd := exec.CommandContext(ctx, g.Config.BinaryPath, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = stdin
	return cmd.Run()
}
