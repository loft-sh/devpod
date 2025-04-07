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

	credentialsReq, err := gitcredentials.Parse(string(raw))
	if err != nil {
		return err
	}

	// try to get the credentials from the workspace server first
	credentials := getCredentialsFromWorkspaceServer(credentialsReq)
	if credentials == nil && cmd.Port != 0 {
		// try to get the credentials from the local machine
		credentials = getCredentialsFromLocalMachine(credentialsReq, cmd.Port)
	}

	// if we still don't have credentials, just return nothing
	if credentials == nil {
		return nil
	}

	// print response to stdout
	fmt.Print(gitcredentials.ToString(credentials))
	return nil
}

func getCredentialsFromWorkspaceServer(credentials *gitcredentials.GitCredentials) *gitcredentials.GitCredentials {
	if _, err := os.Stat(filepath.Join(container.RootDir, ts.RunnerProxySocket)); err != nil {
		// workspace server is not running
		return nil
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", filepath.Join(container.RootDir, ts.RunnerProxySocket))
			},
		},
	}

	credentials, credentialsErr := doRequest(httpClient, credentials, "http://runner-proxy/git-credentials")
	if credentialsErr != nil {
		// append error to /tmp/git-credentials.log
		file, err := os.OpenFile("/tmp/git-credentials-error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil
		}
		defer file.Close()

		_, _ = file.WriteString(fmt.Sprintf("get credentials from workspace server: %v\n", credentialsErr))
		return nil
	}

	return credentials
}

func getCredentialsFromLocalMachine(credentials *gitcredentials.GitCredentials, port int) *gitcredentials.GitCredentials {
	credentials, credentialsErr := doRequest(devpodhttp.GetHTTPClient(), credentials, "http://localhost:"+strconv.Itoa(port)+"/git-credentials")
	if credentialsErr != nil {
		// append error to /tmp/git-credentials.log
		file, err := os.OpenFile("/tmp/git-credentials-error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil
		}
		defer file.Close()

		_, _ = file.WriteString(fmt.Sprintf("get credentials from local machine: %v\n", credentialsErr))
		return nil
	}

	return credentials
}

func doRequest(httpClient *http.Client, credentials *gitcredentials.GitCredentials, url string) (*gitcredentials.GitCredentials, error) {
	rawJSON, err := json.Marshal(credentials)
	if err != nil {
		return nil, fmt.Errorf("error marshalling credentials: %w", err)
	}

	response, err := httpClient.Post(url, "application/json", bytes.NewReader(rawJSON))
	if err != nil {
		return nil, fmt.Errorf("error retrieving credentials from credentials server: %w", err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading credentials: %w", err)
	}

	// has the request succeeded?
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error reading credentials (%d): %s", response.StatusCode, string(raw))
	}

	credentials = &gitcredentials.GitCredentials{}
	err = json.Unmarshal(raw, credentials)
	if err != nil {
		return nil, fmt.Errorf("error decoding credentials: %w", err)
	}

	return credentials, nil
}
