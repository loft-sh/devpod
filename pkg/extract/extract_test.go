package extract

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractSymlinkConflicts(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "extract_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name           string
		setupExisting  func(string) error
		expectedResult string
	}{
		{
			name: "create_new_symlink",
			setupExisting: func(dir string) error {
				// No existing file
				return nil
			},
			expectedResult: "target.txt",
		},
		{
			name: "replace_existing_symlink_different_target",
			setupExisting: func(dir string) error {
				// Create existing symlink with different target
				return os.Symlink("old_target.txt", filepath.Join(dir, "test_symlink"))
			},
			expectedResult: "target.txt",
		},
		{
			name: "preserve_existing_symlink_same_target",
			setupExisting: func(dir string) error {
				// Create existing symlink with same target
				return os.Symlink("target.txt", filepath.Join(dir, "test_symlink"))
			},
			expectedResult: "target.txt",
		},
		{
			name: "replace_existing_regular_file",
			setupExisting: func(dir string) error {
				// Create existing regular file
				return os.WriteFile(filepath.Join(dir, "test_symlink"), []byte("content"), 0644)
			},
			expectedResult: "target.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := filepath.Join(tempDir, tt.name)
			err := os.MkdirAll(testDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create test dir: %v", err)
			}

			// Setup existing file/symlink if needed
			if err := tt.setupExisting(testDir); err != nil {
				t.Fatalf("Failed to setup existing file: %v", err)
			}

			// Create tar archive with symlink
			tarData := createTarWithSymlink(t, "test_symlink", "target.txt")

			// Extract the tar
			err = Extract(bytes.NewReader(tarData), testDir)
			if err != nil {
				t.Fatalf("Extract failed: %v", err)
			}

			// Verify the symlink was created correctly
			symlinkPath := filepath.Join(testDir, "test_symlink")
			linkTarget, err := os.Readlink(symlinkPath)
			if err != nil {
				t.Fatalf("Failed to read symlink: %v", err)
			}

			if linkTarget != tt.expectedResult {
				t.Errorf("Expected symlink target %q, got %q", tt.expectedResult, linkTarget)
			}
		})
	}
}

func TestExtractSymlinkMultipleConflicts(t *testing.T) {
	// Test multiple symlinks pointing to the same target with re-extraction
	tempDir, err := os.MkdirTemp("", "extract_multi_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create the target file first
	targetPath := filepath.Join(tempDir, "target.txt")
	err = os.WriteFile(targetPath, []byte("target content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create target.txt: %v", err)
	}

	// Create existing symlinks (this should be preserved/replaced correctly)
	link1Path := filepath.Join(tempDir, "link1.txt")
	err = os.Symlink("target.txt", link1Path)
	if err != nil {
		t.Fatalf("Failed to create existing link1.txt symlink: %v", err)
	}

	// Create tar archive with the same symlinks (simulating the re-upload scenario)
	tarData := createTarWithMultipleSymlinks(t)

	// Extract should not fail even with existing symlinks
	err = Extract(bytes.NewReader(tarData), tempDir)
	if err != nil {
		t.Fatalf("Extract failed with existing symlinks: %v", err)
	}

	// Verify both symlinks point to target.txt
	link1Target, err := os.Readlink(link1Path)
	if err != nil {
		t.Fatalf("Failed to read link1.txt symlink: %v", err)
	}
	if link1Target != "target.txt" {
		t.Errorf("Expected link1.txt -> target.txt, got %q", link1Target)
	}

	link2Path := filepath.Join(tempDir, "link2.txt")
	link2Target, err := os.Readlink(link2Path)
	if err != nil {
		t.Fatalf("Failed to read link2.txt symlink: %v", err)
	}
	if link2Target != "target.txt" {
		t.Errorf("Expected link2.txt -> target.txt, got %q", link2Target)
	}
}

// Helper function to create a tar archive with a single symlink
func createTarWithSymlink(t *testing.T, linkName, target string) []byte {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Add symlink to tar
	hdr := &tar.Header{
		Name:     linkName,
		Linkname: target,
		Typeflag: tar.TypeSymlink,
		Mode:     0755,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("Failed to write symlink header: %v", err)
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("Failed to close tar writer: %v", err)
	}

	return buf.Bytes()
}

// Helper function to create a tar archive with multiple symlinks
func createTarWithMultipleSymlinks(t *testing.T) []byte {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Add target file
	targetContent := []byte("target content")
	hdr := &tar.Header{
		Name: "target.txt",
		Size: int64(len(targetContent)),
		Mode: 0644,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("Failed to write target.txt header: %v", err)
	}
	if _, err := tw.Write(targetContent); err != nil {
		t.Fatalf("Failed to write target.txt content: %v", err)
	}

	// Add first symlink
	hdr = &tar.Header{
		Name:     "link1.txt",
		Linkname: "target.txt",
		Typeflag: tar.TypeSymlink,
		Mode:     0755,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("Failed to write link1.txt symlink header: %v", err)
	}

	// Add second symlink
	hdr = &tar.Header{
		Name:     "link2.txt",
		Linkname: "target.txt",
		Typeflag: tar.TypeSymlink,
		Mode:     0755,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("Failed to write link2.txt symlink header: %v", err)
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("Failed to close tar writer: %v", err)
	}

	return buf.Bytes()
}

func TestExtractGzippedTarWithSymlinks(t *testing.T) {
	// Test gzipped tar archives with symlinks
	tempDir, err := os.MkdirTemp("", "extract_gzip_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create gzipped tar with symlinks
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Add a file
	content := []byte("test content")
	hdr := &tar.Header{
		Name: "test.txt",
		Size: int64(len(content)),
		Mode: 0644,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("Failed to write file header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("Failed to write file content: %v", err)
	}

	// Add symlink
	hdr = &tar.Header{
		Name:     "link.txt",
		Linkname: "test.txt",
		Typeflag: tar.TypeSymlink,
		Mode:     0755,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("Failed to write symlink header: %v", err)
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("Failed to close tar writer: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("Failed to close gzip writer: %v", err)
	}

	// Extract gzipped tar
	err = Extract(bytes.NewReader(buf.Bytes()), tempDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Verify symlink
	linkPath := filepath.Join(tempDir, "link.txt")
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Failed to read symlink: %v", err)
	}
	if target != "test.txt" {
		t.Errorf("Expected symlink target test.txt, got %q", target)
	}
}
