package helper

import (
	"fmt"
	"io"
	"log"

	"github.com/loft-sh/devpod/pkg/daemon/workspace/network"
	"github.com/spf13/cobra"
)

var requestCmd = &cobra.Command{
	Use:   "request [path]",
	Short: "Send an HTTP request to the specified path via the DevPod network",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		client := network.GetHTTPClient()

		url := fmt.Sprintf("http://%s", path)
		log.Printf("Sending request to %s via DevPod network", url)

		resp, err := client.Get(url)
		if err != nil {
			log.Fatalf("HTTP request error: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("Error reading response: %v", err)
		}
		fmt.Printf("Response:\n%s\n", body)
	},
}
