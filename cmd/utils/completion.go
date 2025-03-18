package utils

import (
	"strings"

	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

func GetWorkspaceSuggestions(rootCmd *cobra.Command, context, provider string, args []string, toComplete string, owner platform.OwnerFilter, logger log.Logger) ([]string, cobra.ShellCompDirective) {
	devPodConfig, err := config.LoadConfig(context, provider)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	workspaces, err := workspace.List(rootCmd.Context(), devPodConfig, false, owner, logger)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var suggestions []string
	for _, ws := range workspaces {
		if strings.HasPrefix(ws.ID, toComplete) {
			suggestions = append(suggestions, ws.ID)
		}
	}
	return suggestions, cobra.ShellCompDirectiveNoFileComp
}
