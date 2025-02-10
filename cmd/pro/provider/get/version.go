package get

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// VersionCmd holds the cmd flags
type VersionCmd struct {
	*flags.GlobalFlags

	Log log.Logger
}

type VersionInfo struct {
	// ServerVersion is the platform deployment version
	ServerVersion string `json:"serverVersion,omitempty"`

	// RemoteProviderVersion is the desired provider version of the current platform deployment
	RemoteProviderVersion string `json:"remoteProviderVersion,omitempty"`

	// CurrentProviderVersion is the currently installed provider version
	CurrentProviderVersion string `json:"currentProviderVersion,omitempty"`
}

// NewVersionCmd creates a new command
func NewVersionCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &VersionCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:   "version",
		Short: "Get platform version",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), os.Stdin, os.Stdout, os.Stderr)
		},
	}

	return c
}

func (cmd *VersionCmd) Run(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	baseClient, err := client.InitClientFromPath(ctx, cmd.Config)
	if err != nil {
		return err
	}

	providerContext := os.Getenv(provider.PROVIDER_CONTEXT)
	if providerContext == "" {
		providerContext = config.DefaultContext
	}
	providerID := os.Getenv(provider.PROVIDER_ID)
	if providerID == "" {
		return fmt.Errorf("provider ID %s not defined", providerID)
	}

	// get our own version
	providerConfig, err := provider.LoadProviderConfig(providerContext, providerID)
	if err != nil {
		return err
	}
	providerVersion := providerConfig.Version

	// get platform version
	platformVersion, err := platform.GetPlatformVersion(baseClient.Config().Host)
	if err != nil {
		return err
	}

	v := VersionInfo{
		ServerVersion:          platformVersion.Version,
		RemoteProviderVersion:  platformVersion.DevPodVersion,
		CurrentProviderVersion: providerVersion,
	}
	out, err := json.Marshal(v)
	if err != nil {
		return err
	}

	fmt.Println(string(out))

	return nil
}
