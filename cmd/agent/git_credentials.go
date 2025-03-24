package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/loft-sh/devpod/cmd/agent/container"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/gitcredentials"
	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/devpod/pkg/ts"
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
			return cmd.Run(context.Background(), args, log.Default.ErrorStreamOnly())
		},
	}
	gitCredentialsCmd.Flags().IntVar(&cmd.Port, "port", 0, "If specified, will use the given port")
	return gitCredentialsCmd
}

func (cmd *GitCredentialsCmd) Run(ctx context.Context, args []string, log log.Logger) error {
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

	// try to get the credentials from the workspace server first
	credentials, err = getCredentialsFromWorkspaceServer(credentials, log)
	if err != nil {
		return err
	} else if credentials == nil && cmd.Port != 0 {
		// try to get the credentials from the local machine
		credentials, err = getCredentialsFromLocalMachine(credentials, cmd.Port, log)
		if err != nil {
			return err
		}
	}

	// if we still don't have credentials, just return nothing
	if credentials == nil {
		return nil
	}

	// print response to stdout
	fmt.Print(gitcredentials.ToString(credentials))
	return nil
}

func getCredentialsFromWorkspaceServer(credentials *gitcredentials.GitCredentials, log log.Logger) (*gitcredentials.GitCredentials, error) {
	if _, err := os.Stat(filepath.Join(container.RootDir, ts.RunnerProxySocket)); err != nil {
		// workspace server is not running
		return nil, nil
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", filepath.Join(container.RootDir, ts.RunnerProxySocket))
			},
		},
	}

	return doRequest(httpClient, credentials, "http://runner-proxy/git-credentials", log)
}

func getCredentialsFromLocalMachine(credentials *gitcredentials.GitCredentials, port int, log log.Logger) (*gitcredentials.GitCredentials, error) {
	return doRequest(devpodhttp.GetHTTPClient(), credentials, "http://localhost:"+strconv.Itoa(port)+"/git-credentials", log)
}

func doRequest(httpClient *http.Client, credentials *gitcredentials.GitCredentials, url string, log log.Logger) (*gitcredentials.GitCredentials, error) {
	rawJSON, err := json.Marshal(credentials)
	if err != nil {
		return nil, err
	}

	response, err := httpClient.Post(url, "application/json", bytes.NewReader(rawJSON))
	if err != nil {
		log.Errorf("Error retrieving credentials from credentials server: %v", err)
		return nil, nil
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Error reading credentials: %v", err)
		return nil, nil
	}

	// has the request succeeded?
	if response.StatusCode != http.StatusOK {
		log.Errorf("Error reading credentials (%d): %v", response.StatusCode, string(raw))
		return nil, nil
	}

	credentials = &gitcredentials.GitCredentials{}
	err = json.Unmarshal(raw, credentials)
	if err != nil {
		log.Errorf("Error decoding credentials: %v", err)
		return nil, nil
	}

	return credentials, nil
}
