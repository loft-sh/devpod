package completion

import (
	"strings"

	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

func GetPlatformHostSuggestions(rootCmd *cobra.Command, context, provider string, args []string, toComplete string, owner platform.OwnerFilter, logger log.Logger) ([]string, cobra.ShellCompDirective) {
	devPodConfig, err := config.LoadConfig(context, provider)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	proInstances, err := workspace.ListProInstances(devPodConfig, logger)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var suggestions []string

	for _, instance := range proInstances {
		if strings.HasPrefix(instance.Host, toComplete) {
			suggestions = append(suggestions, instance.Host)
		}
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}
