package context

import (
	"context"
	"fmt"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// UseCmd holds the use cmd flags
type UseCmd struct {
	flags.GlobalFlags

	Options []string
}

// NewUseCmd uses a new command
func NewUseCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &UseCmd{
		GlobalFlags: *flags,
	}
	useCmd := &cobra.Command{
		Use:   "use",
		Short: "Set a DevPod context as the default",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("please specify the context to use")
			}

			return cmd.Run(context.Background(), args[0])
		},
	}

	useCmd.Flags().StringSliceVarP(&cmd.Options, "option", "o", []string{}, "context option in the form KEY=VALUE")
	return useCmd
}

// Run runs the command logic
func (cmd *UseCmd) Run(ctx context.Context, context string) error {
	devPodConfig, err := config.LoadConfig("", cmd.Provider)
	if err != nil {
		return err
	} else if devPodConfig.Contexts[context] == nil {
		return fmt.Errorf("context '%s' doesn't exist", context)
	}

	// check if there are use options set
	if len(cmd.Options) > 0 {
		err = setOptions(devPodConfig, context, cmd.Options)
		if err != nil {
			return err
		}
	}

	devPodConfig.DefaultContext = context
	err = config.SaveConfig(devPodConfig)
	if err != nil {
		return errors.Wrap(err, "save config")
	}

	return nil
}
