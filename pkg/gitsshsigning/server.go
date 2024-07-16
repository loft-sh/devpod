package gitsshsigning

import (
	"bytes"
	"fmt"
	"os/exec"
)

type GitSSHSignatureRequest struct {
	Content string
	KeyPath string
}

type GitSSHSignatureResponse struct {
	Signature []byte
}

// Sign signs the content using the private key and returns the signature.
// This is intended to be a drop-in replacement for gpg.ssh.program for git,
// so we simply execute ssh-keygen in the same way as git would do locally.
func (req *GitSSHSignatureRequest) Sign() (*GitSSHSignatureResponse, error) {
	// Create a buffer to store the commit content
	var commitBuffer bytes.Buffer
	commitBuffer.WriteString(req.Content)

	// Create the command to run ssh-keygen
	cmd := exec.Command("ssh-keygen", "-Y", "sign", "-f", req.KeyPath, "-n", "git")
	cmd.Stdin = &commitBuffer

	// Capture the output of the command
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to sign commit: %w, stderr: %s", err, stderr.String())
	}

	return &GitSSHSignatureResponse{
		Signature: out.Bytes(),
	}, nil
}
