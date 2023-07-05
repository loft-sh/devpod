package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/gitcredentials"
	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// GitCredentialsCmd holds the cmd flags
type GitCredentialsCmd struct {
	*flags.GlobalFlags

	Port int
}

// NewGitCredentialsCmd creates a new command
func NewGitCredentialsCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &GitCredentialsCmd{
		GlobalFlags: flags,
	}
	gitCredentialsCmd := &cobra.Command{
		Use:   "git-credentials",
		Short: "Retrieves git-credentials from the local machine",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}
	gitCredentialsCmd.Flags().IntVar(&cmd.Port, "port", 0, "If specified, will use the given port")
	_ = gitCredentialsCmd.MarkFlagRequired("port")
	return gitCredentialsCmd
}

func (cmd *GitCredentialsCmd) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return nil
	} else if args[0] != "get" {
		return nil
	}

	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	credentials, err := gitcredentials.Parse(string(raw))
	if err != nil {
		return err
	}

	rawJSON, err := json.Marshal(credentials)
	if err != nil {
		return err
	}

	response, err := devpodhttp.GetHTTPClient().Post("http://localhost:"+strconv.Itoa(cmd.Port)+"/git-credentials", "application/json", bytes.NewReader(rawJSON))
	if err != nil {
		log.Default.ErrorStreamOnly().Errorf("Error retrieving credentials: %v", err)
		return nil
	}
	defer response.Body.Close()

	raw, err = io.ReadAll(response.Body)
	if err != nil {
		log.Default.ErrorStreamOnly().Errorf("Error reading credentials: %v", err)
		return nil
	}

	credentials = &gitcredentials.GitCredentials{}
	err = json.Unmarshal(raw, credentials)
	if err != nil {
		log.Default.ErrorStreamOnly().Errorf("Error decoding credentials: %v", err)
		return nil
	}

	// print response to stdout
	fmt.Print(gitcredentials.ToString(credentials))
	return nil
}
