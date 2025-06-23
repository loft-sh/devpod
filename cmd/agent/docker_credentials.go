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
	"strings"
	"time"

	"github.com/loft-sh/devpod/cmd/agent/container"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/dockercredentials"
	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/devpod/pkg/ts"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// DockerCredentialsCmd holds the cmd flags
type DockerCredentialsCmd struct {
	*flags.GlobalFlags

	Port int
}

// NewDockerCredentialsCmd creates a new command
func NewDockerCredentialsCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &DockerCredentialsCmd{
		GlobalFlags: flags,
	}
	dockerCredentialsCmd := &cobra.Command{
		Use:   "docker-credentials",
		Short: "Retrieves docker-credentials from the local machine",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args, log.Default.ErrorStreamOnly())
		},
	}
	dockerCredentialsCmd.Flags().IntVar(&cmd.Port, "port", 0, "If specified, will use the given port")
	_ = dockerCredentialsCmd.MarkFlagRequired("port")
	return dockerCredentialsCmd
}

func (cmd *DockerCredentialsCmd) Run(ctx context.Context, args []string, log log.Logger) error {
	if len(args) == 0 {
		return nil
	}

	// we only handle get and list
	if args[0] == "get" {
		return cmd.handleGet(log)
	} else if args[0] == "list" {
		return cmd.handleList(log)
	}

	return nil
}

func (cmd *DockerCredentialsCmd) handleList(log log.Logger) error {
	rawJSON, err := json.Marshal(&dockercredentials.Request{})
	if err != nil {
		return err
	}

	response, err := devpodhttp.GetHTTPClient().Post("http://localhost:"+strconv.Itoa(cmd.Port)+"/docker-credentials", "application/json", bytes.NewReader(rawJSON))
	if err != nil {
		log.Errorf("Error retrieving list credentials: %v", err)
		return nil
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Error reading list credentials: %v", err)
		return nil
	}

	// has the request succeeded?
	if response.StatusCode != http.StatusOK {
		log.Errorf("Error reading list credentials (%d): %v", response.StatusCode, string(raw))
		return nil
	}

	listResponse := &dockercredentials.ListResponse{}
	err = json.Unmarshal(raw, listResponse)
	if err != nil {
		log.Errorf("Error decoding list credentials: %s%v", string(raw), err)
		return nil
	}

	if listResponse.Registries == nil {
		listResponse.Registries = map[string]string{}
	}
	raw, err = json.Marshal(listResponse.Registries)
	if err != nil {
		log.Errorf("Error encoding list credentials: %v", err)
		return nil
	}

	// print response to stdout
	fmt.Print(string(raw))
	return nil
}

func (cmd *DockerCredentialsCmd) handleGet(log log.Logger) error {
	url, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	} else if len(strings.TrimSpace(string(url))) == 0 {
		return fmt.Errorf("no credentials server URL")
	}

	credentials := getDockerCredentialsFromWorkspaceServer(&dockercredentials.Credentials{ServerURL: strings.TrimSpace(string(url))})
	if credentials != nil {
		raw, err := json.Marshal(credentials)
		if err != nil {
			log.Errorf("Error encoding credentials: %v", err)
			return nil
		}
		fmt.Print(string(raw))
		return nil
	}

	rawJSON, err := json.Marshal(&dockercredentials.Request{ServerURL: strings.TrimSpace(string(url))})
	if err != nil {
		return err
	}

	response, err := devpodhttp.GetHTTPClient().Post("http://localhost:"+strconv.Itoa(cmd.Port)+"/docker-credentials", "application/json", bytes.NewReader(rawJSON))
	if err != nil {
		log.Errorf("Error retrieving credentials: %v", err)
		return nil
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Error reading credentials: %v", err)
		return nil
	}

	// has the request succeeded?
	if response.StatusCode != http.StatusOK {
		log.Errorf("Error reading credentials (%d): %v", response.StatusCode, string(raw))
		return nil
	}

	// try to unmarshal
	err = json.Unmarshal(raw, &dockercredentials.Credentials{})
	if err != nil {
		log.Errorf("Error parsing credentials: %v", err)
		return nil
	}

	// print response to stdout
	fmt.Print(string(raw))
	return nil
}

func getDockerCredentialsFromWorkspaceServer(credentials *dockercredentials.Credentials) *dockercredentials.Credentials {
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
		Timeout: 15 * time.Second,
	}

	credentials, credentialsErr := requestDockerCredentials(httpClient, credentials, "http://runner-proxy/docker-credentials")
	if credentialsErr != nil {
		// append error to /var/devpod/docker-credentials.log
		file, err := os.OpenFile("/var/devpod/docker-credentials-error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil
		}
		defer file.Close()

		_, _ = file.WriteString(fmt.Sprintf("get credentials from workspace server: %v\n", credentialsErr))
		return nil
	}

	return credentials
}

func requestDockerCredentials(httpClient *http.Client, credentials *dockercredentials.Credentials, url string) (*dockercredentials.Credentials, error) {
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

	credentials = &dockercredentials.Credentials{}
	err = json.Unmarshal(raw, credentials)
	if err != nil {
		return nil, fmt.Errorf("error decoding credentials: %w", err)
	}

	return credentials, nil
}
