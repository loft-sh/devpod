package ide

import (
	"context"
	"fmt"
	"strings"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/devpod/pkg/ide/ideparse"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// UseCmd holds the use cmd flags
type UseCmd struct {
	flags.GlobalFlags

	Options []string
}

// NewUseCmd creates a new command
func NewUseCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &UseCmd{
		GlobalFlags: *flags,
	}
	useCmd := &cobra.Command{
		Use:   "use",
		Short: "Configure the default IDE to use (list available IDEs with 'devpod ide list')",
		Long: `Configure the default IDE to use

Available IDEs can be listed with 'devpod ide list'`,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("please specify the ide to use, list available IDEs with 'devpod ide list'")
			}

			return cmd.Run(context.Background(), args[0])
		},
	}

	useCmd.Flags().StringArrayVarP(&cmd.Options, "option", "o", []string{}, "IDE option in the form KEY=VALUE")
	return useCmd
}

// Run runs the command logic
func (cmd *UseCmd) Run(ctx context.Context, ide string) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	ide = strings.ToLower(ide)
	ideOptions, err := ideparse.GetIDEOptions(ide)
	if err != nil {
		return err
	}

	// check if there are user options set
	if len(cmd.Options) > 0 {
		err = setOptions(devPodConfig, ide, cmd.Options, ideOptions)
		if err != nil {
			return err
		}
	}

	devPodConfig.Current().DefaultIDE = ide
	err = config.SaveConfig(devPodConfig)
	if err != nil {
		return errors.Wrap(err, "save config")
	}

	return nil
}

func setOptions(devPodConfig *config.Config, ide string, options []string, ideOptions ide.Options) error {
	optionValues, err := ideparse.ParseOptions(options, ideOptions)
	if err != nil {
		return err
	}

	if devPodConfig.Current().IDEs == nil {
		devPodConfig.Current().IDEs = map[string]*config.IDEConfig{}
	}

	newValues := map[string]config.OptionValue{}
	if devPodConfig.Current().IDEs[ide] != nil {
		for k, v := range devPodConfig.Current().IDEs[ide].Options {
			newValues[k] = v
		}
	}
	for k, v := range optionValues {
		newValues[k] = v
	}

	devPodConfig.Current().IDEs[ide] = &config.IDEConfig{
		Options: newValues,
	}
	return nil
}
