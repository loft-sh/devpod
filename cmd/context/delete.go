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
			if len(args) > 1 {
				return fmt.Errorf("please specify the context to delete")
			}

			devPodContext := ""
			if len(args) == 1 {
				devPodContext = args[0]
			}

			return cmd.Run(context.Background(), devPodContext)
		},
	}

	return deleteCmd
}

// Run runs the command logic
func (cmd *DeleteCmd) Run(ctx context.Context, context string) error {
	devPodConfig, err := config.LoadConfig(context, cmd.Provider)
	if err != nil {
		return err
	}

	// check for context
	if context == "" {
		context = devPodConfig.DefaultContext
	} else if devPodConfig.Contexts[context] == nil {
		return fmt.Errorf("context '%s' doesn't exist", context)
	}

	// check for default context
	if context == "default" {
		return fmt.Errorf("cannot delete 'default' context")
	}

	delete(devPodConfig.Contexts, context)
	if devPodConfig.DefaultContext == context {
		devPodConfig.DefaultContext = "default"
	}
	if devPodConfig.OriginalContext == context {
		devPodConfig.OriginalContext = "default"
	}

	err = config.SaveConfig(devPodConfig)
	if err != nil {
		return errors.Wrap(err, "save config")
	}

	return nil
}
