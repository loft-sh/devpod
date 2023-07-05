package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/loft-sh/devpod/cmd/flags"
	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/spf13/cobra"
)

// ListAvailableCmd holds the list cmd flags
type ListAvailableCmd struct {
	flags.GlobalFlags
}

func getDevpodProviderList() error {
	req, err := http.NewRequest("GET", "https://api.github.com/users/loft-sh/repos", nil)
	if err != nil {
		return err
	}
	resp, err := devpodhttp.GetHTTPClient().Do(req)
	if err != nil {
		return err
	}

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var jsonResult []map[string]interface{}
	err = json.Unmarshal(result, &jsonResult)
	if err != nil {
		return err
	}

	fmt.Println("List of available providers from loft:")
	for _, v := range jsonResult {
		if strings.Contains(v["name"].(string), "devpod-provider") {
			name := strings.TrimPrefix(v["name"].(string), "devpod-provider-")
			fmt.Println("\t", name)
		}
	}

	return nil
}

// NewListAvailableCmd creates a new command
func NewListAvailableCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &ListAvailableCmd{
		GlobalFlags: *flags,
	}
	listAvailableCmd := &cobra.Command{
		Use:   "list-available",
		Short: "List providers available for installation",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background())
		},
	}

	return listAvailableCmd
}

// Run runs the command logic
func (cmd *ListAvailableCmd) Run(ctx context.Context) error {
	return getDevpodProviderList()
}
