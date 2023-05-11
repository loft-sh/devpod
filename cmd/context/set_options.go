package context

import (
	"context"
	"fmt"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// SetOptionsCmd holds the setOptions cmd flags
type SetOptionsCmd struct {
	flags.GlobalFlags

	Options []string
}

// NewSetOptionsCmd setOptionss a new command
func NewSetOptionsCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &SetOptionsCmd{
		GlobalFlags: *flags,
	}
	setOptionsCmd := &cobra.Command{
		Use:   "set-options",
		Short: "Set options for a DevPod context",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("please specify the context")
			}

			return cmd.Run(context.Background(), args[0])
		},
	}

	setOptionsCmd.Flags().StringSliceVarP(&cmd.Options, "option", "o", []string{}, "context option in the form KEY=VALUE")
	return setOptionsCmd
}

// Run runs the command logic
func (cmd *SetOptionsCmd) Run(ctx context.Context, context string) error {
	devPodConfig, err := config.LoadConfig("", cmd.Provider)
	if err != nil {
		return err
	} else if devPodConfig.Contexts[context] == nil {
		return fmt.Errorf("context '%s' doesn't exist", context)
	}

	// check if there are setOptions options set
	if len(cmd.Options) > 0 {
		err = setOptions(devPodConfig, context, cmd.Options)
		if err != nil {
			return err
		}
	}

	err = config.SaveConfig(devPodConfig)
	if err != nil {
		return errors.Wrap(err, "save config")
	}

	return nil
}
