package context

import (
	"context"
	"fmt"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DeleteCmd holds the delete cmd flags
type DeleteCmd struct {
	flags.GlobalFlags
}

// NewDeleteCmd deletes a new command
func NewDeleteCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &DeleteCmd{
		GlobalFlags: *flags,
	}
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a DevPod context",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("please specify the context to delete")
			}

			return cmd.Run(context.Background(), args[0])
		},
	}

	return deleteCmd
}

// Run runs the command logic
func (cmd *DeleteCmd) Run(ctx context.Context, context string) error {
	devPodConfig, err := config.LoadConfig("", cmd.Provider)
	if err != nil {
		return err
	} else if devPodConfig.Contexts[context] == nil {
		return fmt.Errorf("context '%s' doesn't exist", context)
	}

	delete(devPodConfig.Contexts, context)
	if devPodConfig.DefaultContext == context {
		devPodConfig.DefaultContext = ""
	}

	err = config.SaveConfig(devPodConfig)
	if err != nil {
		return errors.Wrap(err, "save config")
	}

	return nil
}
