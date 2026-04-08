package vscode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/loft-sh/log"
)

func TestFindServerBinaryPathCursorLegacyLayout(t *testing.T) {
	t.Parallel()

	location := t.TempDir()
	expectedPath := filepath.Join(location, "bin", "123456", "bin", "cursor-server")
	writeExecutable(t, expectedPath)

	server := &VsCodeServer{
		flavor: FlavorCursor,
		log:    log.Discard,
	}

	if got := server.findServerBinaryPath(location); got != expectedPath {
		t.Fatalf("expected %q, got %q", expectedPath, got)
	}
}

func TestFindServerBinaryPathCursorPlatformLayout(t *testing.T) {
	t.Parallel()

	location := t.TempDir()
	expectedPath := filepath.Join(location, "bin", "linux-arm64", "abcdef", "bin", "cursor-server")
	writeExecutable(t, expectedPath)
	if err := os.MkdirAll(filepath.Join(location, "bin", "multiplex-server"), 0755); err != nil {
		t.Fatalf("create multiplex-server dir: %v", err)
	}

	server := &VsCodeServer{
		flavor: FlavorCursor,
		log:    log.Discard,
	}

	if got := server.findServerBinaryPath(location); got != expectedPath {
		t.Fatalf("expected %q, got %q", expectedPath, got)
	}
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("create parent dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("write executable: %v", err)
	}
}
