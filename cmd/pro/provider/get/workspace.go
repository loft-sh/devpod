package get

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// WorkspaceCmd holds the cmd flags
type WorkspaceCmd struct {
	*flags.GlobalFlags

	log log.Logger
}

// NewWorkspaceCmd creates a new command
func NewWorkspaceCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &WorkspaceCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:   "workspace",
		Short: "Get workspace for the provider",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	return c
}

func (cmd *WorkspaceCmd) Run(ctx context.Context) error {
	baseClient, err := client.InitClientFromPath(ctx, cmd.Config)
	if err != nil {
		return err
	}

	workspaceInfo, err := platform.GetWorkspaceInfoFromEnv()
	if err != nil {
		return err
	}

	instance, err := platform.FindInstanceInProject(ctx, baseClient, workspaceInfo.UID, workspaceInfo.ProjectName)
	if err != nil {
		return err
	}

	instanceBytes, err := json.Marshal(instance)
	if err != nil {
		return nil
	}

	fmt.Println(string(instanceBytes))

	return nil
}
