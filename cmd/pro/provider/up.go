package provider

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/platform/remotecommand"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
)

// UpCmd holds the cmd flags:
type UpCmd struct {
	*flags.GlobalFlags

	Log     log.Logger
	streams streams
}

type streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// NewUpCmd creates a new command
func NewUpCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &UpCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
		streams: streams{
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		},
	}
	c := &cobra.Command{
		Hidden: true,
		Use:    "up",
		Short:  "Runs up on a workspace",
		Args:   cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	return c
}

func (cmd *UpCmd) Run(ctx context.Context) error {
	baseClient, err := client.InitClientFromPath(ctx, cmd.Config)
	if err != nil {
		return err
	}

	info, err := platform.GetWorkspaceInfoFromEnv()
	if err != nil {
		return err
	}

	instance, err := platform.FindInstanceInProject(ctx, baseClient, info.UID, info.ProjectName)
	if err != nil {
		return err
	}

	return cmd.up(ctx, instance, baseClient)
}

func (cmd *UpCmd) up(ctx context.Context, workspace *managementv1.DevPodWorkspaceInstance, client client.Client) error {
	options := platform.OptionsFromEnv(storagev1.DevPodFlagsUp)
	if options != nil && os.Getenv("DEBUG") == "true" {
		options.Add("debug", "true")
	}

	conn, err := platform.DialInstance(client, workspace, "up", options, cmd.Log)
	if err != nil {
		return err
	}

	_, err = remotecommand.ExecuteConn(ctx, conn, cmd.streams.Stdin, cmd.streams.Stdout, cmd.streams.Stderr, cmd.Log.ErrorStreamOnly())
	if err != nil {
		return fmt.Errorf("error executing: %w", err)
	}

	return nil
}
