package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/credentials"
	"github.com/loft-sh/devpod/pkg/gitsshsigning"
	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

type GitSSHSignatureCmd struct {
	*flags.GlobalFlags

	CertPath   string
	Namespace  string
	BufferFile string
}

func NewGitSSHSignatureCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &GitSSHSignatureCmd{
		GlobalFlags: flags,
	}

	gitSshSignatureCmd := &cobra.Command{
		Use: "git-ssh-signature",
		RunE: func(_ *cobra.Command, args []string) error {
			logger := log.GetInstance()
			if len(args) < 1 {
				logger.Fatalf("Buffer file is required")
			}

			bufferFile := args[len(args)-1]
			content, err := os.ReadFile(bufferFile)
			if err != nil {
				logger.Fatalf("Failed to read content from buffer file: %v", err)
			}

			return cmd.Run(context.Background(), string(content), logger)
		},
	}

	gitSshSignatureCmd.PersistentFlags().StringVarP(&cmd.CertPath, "file", "f", "", "Path to the private key")
	gitSshSignatureCmd.PersistentFlags().StringVarP(&cmd.Namespace, "namespace", "n", "", "Namespace")

	return gitSshSignatureCmd
}

func (cmd *GitSSHSignatureCmd) Run(ctx context.Context, content string, log log.Logger) error {
	log.Infof("::GitSSHSignatureCmd::")
	log.Infof("content: %v", content)
	request := &gitsshsigning.GitSSHSignatureRequest{
		Content: content,
		KeyPath: cmd.CertPath,
	}
	rawJSON, err := json.Marshal(request)
	if err != nil {
		return err
	}

	port, err := credentials.GetPort()
	if err != nil {
		return err
	}

	response, err := devpodhttp.GetHTTPClient().Post(
		"http://localhost:"+strconv.Itoa(port)+"/git-ssh-signature",
		"application/json",
		bytes.NewReader(rawJSON),
	)
	if err != nil {
		log.Errorf("Error retrieving git ssh signature: %v", err)
		return nil
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Error reading git ssh signature: %v", err)
		return nil
	}

	if response.StatusCode != http.StatusOK {
		log.Errorf("Error reading git ssh signature (%d): %v", response.StatusCode, string(raw))
		return nil
	}

	signatureResponse := &gitsshsigning.GitSSHSignatureResponse{}
	err = json.Unmarshal(raw, signatureResponse)
	if err != nil {
		log.Errorf("Error decoding git ssh signature: %v", err)
		return nil
	}

	return nil
}
