package cmd

import (
	"bytes"
	"context"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/pkg/errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var (
	GCLOUD_BINARY = "GCLOUD_BINARY"
	PROJECT       = "PROJECT"
	ZONE          = "ZONE"
	MACHINE_TYPE  = "MACHINE_TYPE"
	DISK_IMAGE    = "DISK_IMAGE"
	DISK_SIZE     = "DISK_SIZE"
)

type ProviderConfig struct {
	BinaryPath string

	MachineType string
	DiskImage   string
	DiskSizeGB  int

	Project string
	Zone    string
}

func ConfigFromEnv() (ProviderConfig, error) {
	diskSize, err := strconv.Atoi(os.Getenv(DISK_SIZE))
	if err != nil {
		return ProviderConfig{}, errors.Wrap(err, "parse disk size")
	}

	return ProviderConfig{
		BinaryPath:  os.Getenv(GCLOUD_BINARY),
		MachineType: os.Getenv(MACHINE_TYPE),
		DiskImage:   os.Getenv(DISK_IMAGE),
		DiskSizeGB:  diskSize,
		Project:     os.Getenv(PROJECT),
		Zone:        os.Getenv(ZONE),
	}, nil
}

func newProvider(log log.Logger) (*gcloudProvider, error) {
	config, err := ConfigFromEnv()
	if err != nil {
		return nil, err
	}

	// create provider
	provider := &gcloudProvider{
		Config: config,
		Log:    log,
	}
	return provider, nil
}

type gcloudProvider struct {
	Config ProviderConfig

	Log              log.Logger
	WorkingDirectory string
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
