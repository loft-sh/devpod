package setup

import (
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/log"
)

func TestChownMounts(t *testing.T) {
	// Create temp directories for mounts
	tempDir := t.TempDir()
	mountTarget1 := filepath.Join(tempDir, "mount1")
	mountTarget2 := filepath.Join(tempDir, "mount2")
	err := os.Mkdir(mountTarget1, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp dir mount1: %v", err)
	}
	err = os.Mkdir(mountTarget2, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp dir mount2: %v", err)
	}

	// Set MarkerBaseDir to temp dir to avoid permission issues
	oldMarkerBaseDir := MarkerBaseDir
	MarkerBaseDir = t.TempDir()
	defer func() { MarkerBaseDir = oldMarkerBaseDir }()

	// Get current user
	currentUser, err := user.Current()
	if err != nil {
		t.Fatalf("Failed to get current user: %v", err)
	}

	// Create a mock result with bind mounts
	result := &config.Result{
		MergedConfig: &config.MergedDevContainerConfig{
			DevContainerConfigBase: config.DevContainerConfigBase{
				RemoteUser: currentUser.Username,
			},
			NonComposeBase: config.NonComposeBase{
				Mounts: []*config.Mount{
					{
						Source: "/local/path",
						Target: mountTarget1,
						Type:   "bind",
					},
				},
			},
		},
		SubstitutionContext: &config.SubstitutionContext{
			WorkspaceMount:           "source=/ws/src,target=" + mountTarget2 + ",type=bind",
			ContainerWorkspaceFolder: mountTarget2,
		},
	}

	// Mock logger
	logger := log.Discard

	// Call ChownMounts
	// We expect it to succeed for the existing directories with current user
	err = ChownMounts(result, logger)
	if err != nil {
		t.Errorf("ChownMounts failed: %v", err)
	}
	
	// Verify marker file created
	markerPath := filepath.Join(MarkerBaseDir, "chownMounts.marker")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Errorf("Marker file not created at %s", markerPath)
	}

	// Call again, should be skipped (log logic inside, but we rely on function returning nil and not erroring)
	err = ChownMounts(result, logger)
	if err != nil {
		t.Errorf("ChownMounts second call failed: %v", err)
	}
}
