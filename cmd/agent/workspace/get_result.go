package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/spf13/cobra"
)

// GetResultCmd holds the cmd flags
type GetResultCmd struct {
	*flags.GlobalFlags

	ID string
}

// NewGetResultCmd creates a new command
func NewGetResultCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &GetResultCmd{
		GlobalFlags: flags,
	}
	getResultCmd := &cobra.Command{
		Use:   "get-result",
		Short: "Returns the devcontainer result",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.Background())
		},
	}
	getResultCmd.Flags().StringVar(&cmd.ID, "id", "", "The workspace id to get result on the agent side")
	_ = getResultCmd.MarkFlagRequired("id")
	return getResultCmd
}

func (cmd *GetResultCmd) Run(ctx context.Context) error {
	// get workspace
	shouldExit, _, err := agent.ReadAgentWorkspaceInfo(cmd.AgentDir, cmd.Context, cmd.ID, log.Default.ErrorStreamOnly())
	if err != nil {
		return err
	} else if shouldExit {
		return nil
	}

	// read dev container result
	result, err := agent.ReadAgentWorkspaceDevContainerResult(cmd.AgentDir, cmd.Context, cmd.ID)
	if err != nil {
		return err
	}

	// return result
	out, err := json.Marshal(result)
	if err != nil {
		return err
	}

	fmt.Print(string(out))
	return nil
}
