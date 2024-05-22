package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/loft"
	"github.com/loft-sh/devpod/pkg/loft/client"
	"github.com/loft-sh/devpod/pkg/loft/remotecommand"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// StatusCmd holds the cmd flags
type StatusCmd struct {
	*flags.GlobalFlags

	Log log.Logger
}

// NewStatusCmd creates a new command
func NewStatusCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &StatusCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Hidden: true,
		Use:    "status",
		Short:  "Runs status on a workspace",
		Args:   cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), os.Stdin, os.Stdout, os.Stderr)
		},
	}

	return c
}

func (cmd *StatusCmd) Run(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	baseClient, err := client.InitClientFromPath(ctx, cmd.Config)
	if err != nil {
		return err
	}

	info, err := GetWorkspaceInfoFromEnv()
	if err != nil {
		return err
	}
	workspace, err := FindWorkspace(ctx, baseClient, info.UID, info.ProjectName)
	if err != nil {
		return err
	} else if workspace == nil {
		out, err := json.Marshal(&storagev1.WorkspaceStatusResult{
			ID:       os.Getenv(loft.WorkspaceIDEnv),
			Context:  os.Getenv(loft.WorkspaceContextEnv),
			State:    string(storagev1.WorkspaceStatusNotFound),
			Provider: os.Getenv(loft.WorkspaceProviderEnv),
		})
		if err != nil {
			return err
		}

		fmt.Println(string(out))
		return nil
	}

	conn, err := DialWorkspace(baseClient, workspace, "getstatus", OptionsFromEnv(storagev1.DevPodFlagsStatus))
	if err != nil {
		return err
	}

	_, err = remotecommand.ExecuteConn(ctx, conn, stdin, stdout, stderr, cmd.Log.ErrorStreamOnly())
	if err != nil {
		return fmt.Errorf("error executing: %w", err)
	}

	return nil
}
