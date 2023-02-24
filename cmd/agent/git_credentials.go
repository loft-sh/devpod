package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/gitcredentials"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"os"
	"strconv"
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

	rawJson, err := json.Marshal(credentials)
	if err != nil {
		return err
	}

	response, err := http.Post("http://localhost:"+strconv.Itoa(cmd.Port)+"/git-credentials", "application/json", bytes.NewReader(rawJson))
	if err != nil {
		return err
	}
	defer response.Body.Close()

	raw, err = io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	credentials = &gitcredentials.GitCredentials{}
	err = json.Unmarshal(raw, credentials)
	if err != nil {
		return errors.Wrapf(err, "decode response %s", string(raw))
	}

	// print response to stdout
	fmt.Print(gitcredentials.ToString(credentials))
	return nil
}
