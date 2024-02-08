package pro

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/binaries"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/spf13/cobra"
)

// ListCmd holds the list cmd flags
type ListCmd struct {
	flags.GlobalFlags

	Output string
	Login  bool
}

// NewListCmd creates a new command
func NewListCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &ListCmd{
		GlobalFlags: *flags,
	}
	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List available pro instances",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background())
		},
	}

	listCmd.Flags().StringVar(&cmd.Output, "output", "plain", "The output format to use. Can be json or plain")
	listCmd.Flags().BoolVar(&cmd.Login, "login", false, "Check if the user is logged into the pro instance")
	return listCmd
}

// Run runs the command logic
func (cmd *ListCmd) Run(ctx context.Context) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	proInstances, err := workspace.ListProInstances(devPodConfig, log.Default)
	if err != nil {
		return err
	}

	if cmd.Output == "plain" {
		tableEntries := [][]string{}
		for _, proInstance := range proInstances {
			entry := []string{
				proInstance.Host,
				proInstance.Provider,
				time.Since(proInstance.CreationTimestamp.Time).Round(1 * time.Second).String(),
			}
			if cmd.Login {
				err = checkLogin(ctx, devPodConfig, proInstance)
				entry = append(entry, fmt.Sprintf("%t", err == nil))
			}

			tableEntries = append(tableEntries, entry)
		}
		sort.SliceStable(tableEntries, func(i, j int) bool {
			return tableEntries[i][0] < tableEntries[j][0]
		})

		tableHeaders := []string{
			"Host",
			"Provider",
			"Age",
		}
		if cmd.Login {
			tableHeaders = append(tableHeaders, "Authenticated")
		}

		table.PrintTable(log.Default, tableHeaders, tableEntries)
	} else if cmd.Output == "json" {
		tableEntries := []*proTableEntry{}
		for _, proInstance := range proInstances {
			entry := &proTableEntry{ProInstance: proInstance}
			if cmd.Login {
				err = checkLogin(ctx, devPodConfig, proInstance)
				isAuthenticated := err == nil
				entry.Authenticated = &isAuthenticated
			}

			tableEntries = append(tableEntries, entry)
		}

		sort.SliceStable(tableEntries, func(i, j int) bool {
			return tableEntries[i].Host < tableEntries[j].Host
		})
		out, err := json.Marshal(tableEntries)
		if err != nil {
			return err
		}
		fmt.Print(string(out))
	} else {
		return fmt.Errorf("unexpected output format, choose either json or plain. Got %s", cmd.Output)
	}

	return nil
}

type proTableEntry struct {
	*provider.ProInstance

	Authenticated *bool `json:"authenticated,omitempty"`
}

func checkLogin(ctx context.Context, devPodConfig *config.Config, proInstance *provider.ProInstance) error {
	providerConfig, err := provider.LoadProviderConfig(devPodConfig.DefaultContext, proInstance.Provider)
	if err != nil {
		return err
	}

	providerBinaries, err := binaries.GetBinaries(devPodConfig.DefaultContext, providerConfig)
	if err != nil {
		return fmt.Errorf("get provider binaries: %w", err)
	} else if providerBinaries[LOFT_PROVIDER_BINARY] == "" {
		return fmt.Errorf("provider is missing %s binary", LOFT_PROVIDER_BINARY)
	}

	providerDir, err := provider.GetProviderDir(devPodConfig.DefaultContext, providerConfig.Name)
	if err != nil {
		return err
	}

	args := []string{
		"login",
		"--log-output=raw",
	}

	extraEnv := []string{
		"LOFT_SKIP_VERSION_CHECK=true",
		"LOFT_CONFIG=" + filepath.Join(providerDir, "loft-config.json"),
	}

	stdout := &bytes.Buffer{}

	// start the command
	loginCmd := exec.CommandContext(ctx, providerBinaries[LOFT_PROVIDER_BINARY], args...)
	loginCmd.Env = os.Environ()
	loginCmd.Env = append(loginCmd.Env, extraEnv...)
	loginCmd.Stdout = stdout
	err = loginCmd.Run()
	if err != nil {
		return fmt.Errorf("run login command: %w", err)
	}

	if stdout.Len() > 0 && strings.Contains(stdout.String(), "Not logged in") {
		return fmt.Errorf("not logged into %s", proInstance.Host)
	}

	return nil
}
