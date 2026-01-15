package gitsshsigning

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSign_WithoutAgent(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gitsshsigning-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Generate a test key pair
	keyPath := filepath.Join(tmpDir, "test_key")
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "", "-q")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	// Test signing without UseAgent (direct private key)
	req := &GitSSHSignatureRequest{
		Content:  "test commit content\n",
		KeyPath:  keyPath,
		UseAgent: false,
	}

	resp, err := req.Sign()
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}

	if len(resp.Signature) == 0 {
		t.Error("Expected non-empty signature")
	}

	// Verify signature looks like an SSH signature
	sigStr := string(resp.Signature)
	if !strings.Contains(sigStr, "-----BEGIN SSH SIGNATURE-----") {
		t.Errorf("Signature doesn't look like SSH signature: %s", sigStr)
	}
}

func TestSign_WithAgent(t *testing.T) {
	// Skip if SSH_AUTH_SOCK is not set (no agent running)
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		t.Skip("Skipping agent test: SSH_AUTH_SOCK not set")
	}

	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gitsshsigning-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Generate a test key pair
	keyPath := filepath.Join(tmpDir, "test_key")
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "", "-q")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	// Add key to agent
	cmd = exec.Command("ssh-add", keyPath)
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to add key to agent (agent may not be running): %v", err)
	}

	// Remove key from agent when done
	defer func() {
		_ = exec.Command("ssh-add", "-d", keyPath).Run()
	}()

	// Test signing with UseAgent (public key + agent lookup)
	req := &GitSSHSignatureRequest{
		Content:  "test commit content\n",
		KeyPath:  keyPath + ".pub", // Use public key path
		UseAgent: true,
	}

	resp, err := req.Sign()
	if err != nil {
		t.Fatalf("Sign() with UseAgent=true failed: %v", err)
	}

	if len(resp.Signature) == 0 {
		t.Error("Expected non-empty signature")
	}

	// Verify signature looks like an SSH signature
	sigStr := string(resp.Signature)
	if !strings.Contains(sigStr, "-----BEGIN SSH SIGNATURE-----") {
		t.Errorf("Signature doesn't look like SSH signature: %s", sigStr)
	}
}

func TestSign_WithAgent_NoAgentFails(t *testing.T) {
	// This test verifies that using UseAgent=true with a public key
	// fails when no agent has the corresponding private key

	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gitsshsigning-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Generate a test key pair (not added to any agent)
	keyPath := filepath.Join(tmpDir, "test_key")
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "", "-q")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	// Temporarily unset SSH_AUTH_SOCK to ensure no agent
	oldSock := os.Getenv("SSH_AUTH_SOCK")
	_ = os.Unsetenv("SSH_AUTH_SOCK")
	defer func() { _ = os.Setenv("SSH_AUTH_SOCK", oldSock) }()

	// Test signing with UseAgent but no agent available
	req := &GitSSHSignatureRequest{
		Content:  "test commit content\n",
		KeyPath:  keyPath + ".pub",
		UseAgent: true,
	}

	_, err = req.Sign()
	if err == nil {
		t.Error("Expected Sign() to fail when UseAgent=true but no agent available")
	}
}
