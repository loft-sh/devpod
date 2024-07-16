package gitsshsigning

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/loft-sh/devpod/pkg/credentials"
	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/log"
)

// extractContentFromGitBuffer extracts content from buffer passed by git for signature
func extractContentFromGitBuffer(bufferFile string) ([]byte, error) {
	content, err := os.ReadFile(bufferFile)
	if err != nil {
		return nil, err
	}

	return content, nil
}

// requestContentSignature handles http request for ssh signature to credentials server
func requestContentSignature(content []byte, certPath, _ string, log log.Logger) ([]byte, error) {
	request := &GitSSHSignatureRequest{
		Content: string(content),
		KeyPath: certPath,
	}
	rawJSON, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	port, err := credentials.GetPort()
	if err != nil {
		return nil, err
	}

	response, err := devpodhttp.GetHTTPClient().Post(
		"http://localhost:"+strconv.Itoa(port)+"/git-ssh-signature",
		"application/json",
		bytes.NewReader(rawJSON),
	)
	if err != nil {
		log.Errorf("Error retrieving git ssh signature: %v", err)
		return nil, err
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Error reading git ssh signature: %v", err)
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf("error reading git ssh signature (%d): %v", response.StatusCode, string(raw))
		return nil, err
	}

	signatureResponse := &GitSSHSignatureResponse{}
	err = json.Unmarshal(raw, signatureResponse)
	if err != nil {
		log.Errorf("Error decoding git ssh signature: %v", err)
		return nil, err
	}

	return signatureResponse.Signature, nil
}

// writeSignatureToFile writes the signed content to the .sig file
func writeSignatureToFile(signature []byte, bufferFile string, log log.Logger) error {
	sigFile := bufferFile + ".sig"
	err := os.WriteFile(sigFile, signature, 0644)
	if err != nil {
		log.Fatalf("Failed to write signature to file: %v", err)
		return err
	}
	return nil
}

// HandleGitSSHProgramCall implements logic handling call from git when signing a commit
func HandleGitSSHProgramCall(certPath, namespace, bufferFile string, log log.Logger) error {
	content, err := extractContentFromGitBuffer(bufferFile)
	if err != nil {
		return err
	}

	signature, err := requestContentSignature(content, certPath, namespace, log)
	if err != nil {
		return err
	}

	err = writeSignatureToFile(signature, bufferFile, log)
	if err != nil {
		return err
	}

	return nil
}
