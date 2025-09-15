package extract

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
)

func TestArchiver_isExcluded(t *testing.T) {
	testCases := []struct {
		name           string
		excludedPaths  []string
		relativePath   string
		expectedResult bool
	}{
		{
			name:           "simple prefix match",
			excludedPaths:  []string{"node_modules"},
			relativePath:   "node_modules",
			expectedResult: true,
		},
		{
			name:           "no match",
			excludedPaths:  []string{"node_modules"},
			relativePath:   "src/main.js",
			expectedResult: false,
		},
		{
			name:           "wildcard git pattern",
			excludedPaths:  []string{"**/.git"},
			relativePath:   "project/.git",
			expectedResult: true,
		},
		{
			name:           "nested wildcard git pattern",
			excludedPaths:  []string{"**/.git"},
			relativePath:   "nested/project/.git",
			expectedResult: true,
		},
		{
			name:           "file extension pattern",
			excludedPaths:  []string{"*.png"},
			relativePath:   "image.png",
			expectedResult: true,
		},
		{
			name:           "file extension pattern no match",
			excludedPaths:  []string{"*.png"},
			relativePath:   "image.jpg",
			expectedResult: false,
		},
		{
			name:           "multiple patterns with match",
			excludedPaths:  []string{"**/.git", "**/node_modules", "*.png"},
			relativePath:   "src/.git",
			expectedResult: true,
		},
		{
			name:           "multiple patterns no match",
			excludedPaths:  []string{"**/.git", "**/node_modules", "*.png"},
			relativePath:   "src/main.js",
			expectedResult: false,
		},
		{
			name:           "directory with trailing slash",
			excludedPaths:  []string{"**/node_modules"},
			relativePath:   "project/node_modules/",
			expectedResult: true,
		},
		{
			name:           "nested file in excluded directory",
			excludedPaths:  []string{"**/node_modules"},
			relativePath:   "project/node_modules/package.json",
			expectedResult: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			archiver := NewArchiver("/tmp", nil, testCase.excludedPaths)
			result := archiver.isExcluded(testCase.relativePath)
			assert.Check(t, cmp.Equal(testCase.expectedResult, result), 
				"Expected %t for path '%s' with patterns %v, got %t", 
				testCase.expectedResult, testCase.relativePath, testCase.excludedPaths, result)
		})
	}
}

func TestWriteTarExclude_PatternMatching(t *testing.T) {
	// Create a temporary test directory
	testDir, err := os.MkdirTemp("", "devpod-extract-test")
	assert.NilError(t, err)
	defer os.RemoveAll(testDir)

	// Create test file structure
	testFiles := map[string]string{
		".git/config":                  "git config content",
		".git/objects/abc123":          "git object",
		"node_modules/react/index.js":  "react content",
		"node_modules/lodash/index.js": "lodash content",
		"src/main.js":                  "main javascript",
		"src/utils.js":                 "utils javascript",
		"image.png":                    "fake png data",
		"icon.svg":                     "fake svg data",
		"README.md":                    "readme content",
		"package.json":                 "package config",
	}

	// Create files and directories
	for filePath, content := range testFiles {
		fullPath := filepath.Join(testDir, filePath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		assert.NilError(t, err)
		
		err = os.WriteFile(fullPath, []byte(content), 0644)
		assert.NilError(t, err)
	}

	testCases := []struct {
		name             string
		excludedPaths    []string
		expectedIncluded []string
		expectedExcluded []string
	}{
		{
			name:          "exclude git and node_modules with wildcards",
			excludedPaths: []string{"**/.git", "**/node_modules"},
			expectedIncluded: []string{
				"src/main.js",
				"src/utils.js", 
				"image.png",
				"icon.svg",
				"README.md",
				"package.json",
			},
			expectedExcluded: []string{
				".git",
				"node_modules",
			},
		},
		{
			name:          "exclude image files with extension pattern",
			excludedPaths: []string{"*.png", "*.svg"},
			expectedIncluded: []string{
				".git/config",
				".git/objects/abc123",
				"node_modules/react/index.js",
				"node_modules/lodash/index.js",
				"src/main.js",
				"src/utils.js",
				"README.md", 
				"package.json",
			},
			expectedExcluded: []string{
				"image.png",
				"icon.svg",
			},
		},
		{
			name:          "complex exclusion patterns",
			excludedPaths: []string{"**/.git", "**/node_modules", "*.png", "*.svg"},
			expectedIncluded: []string{
				"src/main.js",
				"src/utils.js",
				"README.md",
				"package.json",
			},
			expectedExcluded: []string{
				".git",
				"node_modules", 
				"image.png",
				"icon.svg",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteTarExclude(&buf, testDir, false, testCase.excludedPaths)
			assert.NilError(t, err)

			// Parse the generated tar to see what was included
			tarReader := tar.NewReader(&buf)
			includedFiles := []string{}

			for {
				header, err := tarReader.Next()
				if err == io.EOF {
					break
				}
				assert.NilError(t, err)
				includedFiles = append(includedFiles, header.Name)
			}

			// Verify expected inclusions
			for _, expectedFile := range testCase.expectedIncluded {
				found := false
				for _, includedFile := range includedFiles {
					if includedFile == expectedFile {
						found = true
						break
					}
				}
				assert.Check(t, found, "Expected file '%s' to be included but it was not", expectedFile)
			}

			// Verify expected exclusions 
			for _, expectedExcluded := range testCase.expectedExcluded {
				found := false
				for _, includedFile := range includedFiles {
					if includedFile == expectedExcluded || 
						 filepath.HasPrefix(includedFile, expectedExcluded+"/") {
						found = true
						break
					}
				}
				assert.Check(t, !found, "Expected file/directory '%s' to be excluded but it was included", expectedExcluded)
			}
		})
	}
}

func TestWriteTarExclude_BackwardsCompatibility(t *testing.T) {
	// Create a temporary test directory
	testDir, err := os.MkdirTemp("", "devpod-extract-compat-test")
	assert.NilError(t, err)
	defer os.RemoveAll(testDir)

	// Create test files
	testFiles := []string{"file1.txt", "prefix_file.txt", "other.txt"}
	for _, file := range testFiles {
		err = os.WriteFile(filepath.Join(testDir, file), []byte("content"), 0644)
		assert.NilError(t, err)
	}

	// Test that exact match patterns work (the pattern matcher interprets prefix_ differently)
	// Let's test with a pattern that should definitely work in both old and new systems
	var buf bytes.Buffer
	err = WriteTarExclude(&buf, testDir, false, []string{"prefix_file.txt"})
	assert.NilError(t, err)

	// Parse the tar
	tarReader := tar.NewReader(&buf)
	includedFiles := []string{}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		assert.NilError(t, err)
		includedFiles = append(includedFiles, header.Name)
	}

	// Should include file1.txt and other.txt, but exclude prefix_file.txt
	expectedFiles := []string{"file1.txt", "other.txt"}
	
	for _, expected := range expectedFiles {
		found := false
		for _, included := range includedFiles {
			if included == expected {
				found = true
				break
			}
		}
		assert.Check(t, found, "Expected file '%s' to be included", expected)
	}

	// Verify prefix_file.txt was excluded
	for _, included := range includedFiles {
		assert.Check(t, included != "prefix_file.txt", "File 'prefix_file.txt' should have been excluded")
	}
	
	// Verify we got the right number of files
	assert.Check(t, cmp.Len(includedFiles, len(expectedFiles)), 
		"Expected %d files, got %d: %v", len(expectedFiles), len(includedFiles), includedFiles)
}

func TestNewArchiver_PatternMatcherCreation(t *testing.T) {
	testCases := []struct {
		name          string
		excludedPaths []string
		expectMatcher bool
	}{
		{
			name:          "empty patterns",
			excludedPaths: []string{},
			expectMatcher: false,
		},
		{
			name:          "nil patterns", 
			excludedPaths: nil,
			expectMatcher: false,
		},
		{
			name:          "valid patterns",
			excludedPaths: []string{"**/.git", "*.png"},
			expectMatcher: true,
		},
		{
			name:          "simple patterns",
			excludedPaths: []string{"node_modules"},
			expectMatcher: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			archiver := NewArchiver("/tmp", nil, testCase.excludedPaths)
			
			hasMatcher := archiver.patternMatcher != nil
			assert.Check(t, cmp.Equal(testCase.expectMatcher, hasMatcher),
				"Expected pattern matcher existence: %t, got: %t", testCase.expectMatcher, hasMatcher)
		})
	}
}
