package pro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/remotecommand"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

const AllWorkspaces = "all"

// RebuildCmd holds the cmd flags
type RebuildCmd struct {
	*flags.GlobalFlags
	Log log.Logger

	Project string
	Host    string
}

// NewRebuildCmd creates a new command
func NewRebuildCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &RebuildCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:   "rebuild",
		Short: "Rebuild a workspace",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			log.Default.SetFormat(log.TextFormat)

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	c.Flags().StringVar(&cmd.Project, "project", "", "The project to use")
	_ = c.MarkFlagRequired("project")
	c.Flags().StringVar(&cmd.Host, "host", "", "The pro instance to use")
	_ = c.MarkFlagRequired("host")

	return c
}

func (cmd *RebuildCmd) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("please provide a workspace name")
	}
	targetWorkspace := args[0]

	devPodConfig, err := config.LoadConfig(cmd.Context, "")
	if err != nil {
		return err
	}

	baseClient, err := platform.InitClientFromHost(ctx, devPodConfig, cmd.Host, cmd.Log)
	if err != nil {
		return fmt.Errorf("resolve host \"%s\": %w", cmd.Host, err)
	}

	workspace, err := platform.FindInstanceByName(ctx, baseClient, targetWorkspace, cmd.Project)
	if err != nil {
		return err
	}

	opts := struct {
		Recreate bool `json:"recreate"`
	}{Recreate: true}
	rawOpts, err := json.Marshal(opts)
	if err != nil {
		return err
	}
	values := url.Values{"options": []string{string(rawOpts)}, "cliMode": []string{"true"}}
	conn, err := platform.DialInstance(baseClient, workspace, "up", values, cmd.Log)
	if err != nil {
		return err
	}

	_, err = remotecommand.ExecuteConn(ctx, conn, os.Stdin, os.Stdout, os.Stderr, cmd.Log.ErrorStreamOnly())
	if err != nil {
		return fmt.Errorf("error executing: %w", err)
	}

	return nil
}
