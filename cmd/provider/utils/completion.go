package utils

import (
	"strings"

	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

func GetProviderSuggestions(rootCmd *cobra.Command, context, provider string, args []string, toComplete string, owner platform.OwnerFilter, logger log.Logger) ([]string, cobra.ShellCompDirective) {
	devPodConfig, err := config.LoadConfig(context, provider)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	providers, err := workspace.LoadAllProviders(devPodConfig, log.Default.ErrorStreamOnly())
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var suggestions []string
	for _, provider := range providers {
		if strings.HasPrefix(provider.Config.Name, toComplete) {
			suggestions = append(suggestions, provider.Config.Name)
		}
	}
	return suggestions, cobra.ShellCompDirectiveNoFileComp
}
