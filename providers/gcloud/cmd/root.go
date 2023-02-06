package cmd

import "github.com/spf13/cobra"

// NewGCloudCmd returns a new root command
func NewGCloudCmd() *cobra.Command {
	gcloudCmd := &cobra.Command{
		Use:   "gcloud",
		Short: "gcloud Provider commands",
	}

	gcloudCmd.AddCommand(NewCreateCmd())
	gcloudCmd.AddCommand(NewDeleteCmd())
	gcloudCmd.AddCommand(NewCommandCmd())
	gcloudCmd.AddCommand(NewStartCmd())
	gcloudCmd.AddCommand(NewStopCmd())
	gcloudCmd.AddCommand(NewStatusCmd())
	return gcloudCmd
}
