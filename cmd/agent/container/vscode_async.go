package container

import (
	"encoding/json"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/ide/vscode"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// VSCodeAsyncCmd holds the cmd flags
type VSCodeAsyncCmd struct {
	*flags.GlobalFlags

	SetupInfo string
	Flavor    string
}

// NewVSCodeAsyncCmd creates a new command
func NewVSCodeAsyncCmd() *cobra.Command {
	cmd := &VSCodeAsyncCmd{}
	vsCodeAsyncCmd := &cobra.Command{
		Use:   "vscode-async",
		Short: "Starts vscode",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}
	vsCodeAsyncCmd.Flags().StringVar(&cmd.SetupInfo, "setup-info", "", "The container setup info")
	_ = vsCodeAsyncCmd.MarkFlagRequired("setup-info")

	vsCodeAsyncCmd.Flags().StringVar(&cmd.Flavor, "flavor", string(vscode.FlavorStable), "The flavor of the VSCode distribution")
	vsCodeAsyncCmd.Flags().StringVar(&cmd.Flavor, "release-channel", string(vscode.FlavorStable), "The release channel to use for vscode")
	_ = vsCodeAsyncCmd.Flags().MarkDeprecated("release-channel", "prefer the --flavor flag")
	// gracefully migrate --release-channel to --flavor
	vsCodeAsyncCmd.Flags().SetNormalizeFunc(migrateReleaseChannel)
	return vsCodeAsyncCmd
}

func migrateReleaseChannel(f *pflag.FlagSet, name string) pflag.NormalizedName {
	if name == "release-channel" {
		name = "flavor"
	}

	return pflag.NormalizedName(name)
}

// Run runs the command logic
func (cmd *VSCodeAsyncCmd) Run(_ *cobra.Command, _ []string) error {
	log.Default.Debugf("Start setting up container...")
	decompressed, err := compress.Decompress(cmd.SetupInfo)
	if err != nil {
		return err
	}

	setupInfo := &config.Result{}
	err = json.Unmarshal([]byte(decompressed), setupInfo)
	if err != nil {
		return err
	}

	// install IDE
	err = setupVSCodeExtensions(setupInfo, vscode.Flavor(cmd.Flavor), log.Default)
	if err != nil {
		return err
	}

	return nil
}

func setupVSCodeExtensions(setupInfo *config.Result, flavor vscode.Flavor, log log.Logger) error {
	vsCodeConfiguration := config.GetVSCodeConfiguration(setupInfo.MergedConfig)
	user := config.GetRemoteUser(setupInfo)
	return vscode.NewVSCodeServer(vsCodeConfiguration.Extensions, "", user, nil, flavor, log).InstallExtensions()
}
