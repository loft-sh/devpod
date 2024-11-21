package provider

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/platform/remotecommand"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// SshCmd holds the cmd flags
type SshCmd struct {
	*flags.GlobalFlags

	Log log.Logger
}

// NewSshCmd creates a new command
func NewSshCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &SshCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Hidden: true,
		Use:    "ssh",
		Short:  "Runs ssh on a workspace",
		Args:   cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), os.Stdin, os.Stdout, os.Stderr)
		},
	}

	return c
}

func (cmd *SshCmd) Run(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	baseClient, err := client.InitClientFromPath(ctx, cmd.Config)
	if err != nil {
		return err
	}

	info, err := platform.GetWorkspaceInfoFromEnv()
	if err != nil {
		return err
	}
	workspace, err := platform.FindInstanceInProject(ctx, baseClient, info.UID, info.ProjectName)
	if err != nil {
		return err
	} else if workspace == nil {
		return fmt.Errorf("couldn't find workspace")
	}

	opts := platform.OptionsFromEnv(storagev1.DevPodFlagsSsh)
	if os.Getenv("LOFT_TRACE_ID") != "" {
		opts.Add("LOFT_TRACE_ID", os.Getenv("LOFT_TRACE_ID"))
	}
	os.WriteFile("/tmp/loft-ssh-debug", []byte(os.Getenv("LOFT_TRACE_ID")), 0777)

	conn, err := platform.DialInstance(baseClient, workspace, "ssh", opts, cmd.Log)
	if err != nil {
		return err
	}

	start := time.Now()
	defer func() {
		cmd.Log.Infof("pro provider took %dms", time.Since(start).Milliseconds())
	}()

	_, err = remotecommand.ExecuteConn(ctx, conn, stdin, stdout, stderr, cmd.Log.ErrorStreamOnly())
	if err != nil {
		return fmt.Errorf("error executing: %w", err)
	}

	return nil
}
