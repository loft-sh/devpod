package agent

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/loft-sh/devpod/cmd/flags"
	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/devpod/pkg/loftconfig"
	"github.com/loft-sh/log"

	"github.com/spf13/cobra"
)

type SetupLoftPlatformAccessCmd struct {
	*flags.GlobalFlags

	Context  string
	Provider string
	Port     int
}

// NewSetupLoftPlatformAccessCmd creates a new setup-loft-platform-access command
// This agent command can be used to inject loft platform configuration from local machine to workspace.
func NewSetupLoftPlatformAccessCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &SetupLoftPlatformAccessCmd{
		GlobalFlags: flags,
	}

	setupLoftPlatformAccessCmd := &cobra.Command{
		Use:   "setup-loft-platform-access",
		Short: "used to setup Loft Platform access",
		RunE:  cmd.Run,
	}

	setupLoftPlatformAccessCmd.Flags().StringVar(&cmd.Context, "context", "", "context to use")
	setupLoftPlatformAccessCmd.Flags().StringVar(&cmd.Provider, "provider", "", "provider to use")
	setupLoftPlatformAccessCmd.Flags().IntVar(&cmd.Port, "port", 0, "If specified, will use the given port")

	return setupLoftPlatformAccessCmd
}

// Run executes main command logic.
// It fetches Loft Platform credentials from credentials server and sets it up inside the workspace.
func (c *SetupLoftPlatformAccessCmd) Run(_ *cobra.Command, args []string) error {
	logger := log.Default.ErrorStreamOnly()

	request := &loftconfig.LoftConfigRequest{
		Context:  c.Context,
		Provider: c.Provider,
	}

	rawJson, err := json.Marshal(request)
	if err != nil {
		logger.Errorf("Error parsing request: %w", err)
		return err
	}

	response, err := devpodhttp.GetHTTPClient().Post(
		"http://localhost:"+strconv.Itoa(c.Port)+"/loft-platform-credentials",
		"application/json",
		bytes.NewReader(rawJson),
	)
	if err != nil {
		logger.Errorf("Error retrieving credentials: %v", err)
		return err
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		logger.Errorf("Error reading loft config: %w", err)
		return err
	}

	// has the request succeeded?
	if response.StatusCode != http.StatusOK {
		logger.Errorf("Error reading loft config (%d): %w", response.StatusCode, string(raw))
		return err
	}

	configResponse := &loftconfig.LoftConfigResponse{}
	err = json.Unmarshal(raw, configResponse)
	if err != nil {
		logger.Errorf("Error decoding loft config: %s %w", string(raw), err)
		return nil
	}

	configPath, err := loftconfig.LoftConfigPath(c.Context, c.Provider)
	if err != nil {
		return err
	}

	err = loftconfig.StoreLoftConfig(configResponse, configPath)
	if err != nil {
		logger.Errorf("Failed saving loft config: %w", err)
		return err
	}

	return nil
}
