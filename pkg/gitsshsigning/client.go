package gitsshsigning

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strconv"

	"github.com/loft-sh/devpod/pkg/credentials"
	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/log"
)

// HandleGitSSHProgramCall implements logic handling call from git when signing a commit
func HandleGitSSHProgramCall(certPath, namespace, bufferFile string, log log.Logger) error {
	content, err := extractContentFromGitBuffer(bufferFile)
	if err != nil {
		return err
	}

	signature, err := requestContentSignature(content, certPath, log)
	if err != nil {
		return err
	}

	if err := writeSignatureToFile(signature, bufferFile, log); err != nil {
		return err
	}

	return nil
}

// extractContentFromGitBuffer reads the content from the buffer file created by git
func extractContentFromGitBuffer(bufferFile string) ([]byte, error) {
	return os.ReadFile(bufferFile)
}

// requestContentSignature sends an HTTP request to the credentials server to sign the content
func requestContentSignature(content []byte, certPath string, log log.Logger) ([]byte, error) {
	requestBody, err := createSignatureRequestBody(content, certPath)
	if err != nil {
		return nil, err
	}

	responseBody, err := sendSignatureRequest(requestBody, log)
	if err != nil {
		return nil, err
	}

	return parseSignatureResponse(responseBody, log)
}

// writeSignatureToFile writes the signed content to a .sig file
func writeSignatureToFile(signature []byte, bufferFile string, log log.Logger) error {
	sigFile := bufferFile + ".sig"
	if err := os.WriteFile(sigFile, signature, 0644); err != nil {
		log.Fatalf("Failed to write signature to file: %v", err)
		return err
	}
	return nil
}

func createSignatureRequestBody(content []byte, certPath string) ([]byte, error) {
	request := &GitSSHSignatureRequest{
		Content: string(content),
		KeyPath: certPath,
	}
	return json.Marshal(request)
}

func sendSignatureRequest(requestBody []byte, log log.Logger) ([]byte, error) {
	port, err := credentials.GetPort()
	if err != nil {
		return nil, err
	}

	response, err := devpodhttp.GetHTTPClient().Post(
		"http://localhost:"+strconv.Itoa(port)+"/git-ssh-signature", // TODO: build the url, don't hardcode localhost
		"application/json",
		bytes.NewReader(requestBody),
	)
	if err != nil {
		log.Errorf("Error retrieving git ssh signature: %v", err)
		return nil, err
	}
	defer response.Body.Close()

	return io.ReadAll(response.Body)
}

func parseSignatureResponse(responseBody []byte, log log.Logger) ([]byte, error) {
	signatureResponse := &GitSSHSignatureResponse{}
	if err := json.Unmarshal(responseBody, signatureResponse); err != nil {
		log.Errorf("Error decoding git ssh signature: %v", err)
		return nil, err
	}

	return signatureResponse.Signature, nil
}
