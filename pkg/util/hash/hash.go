package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/docker/docker/pkg/longpath"
	"github.com/moby/patternmatcher"
	"github.com/pkg/errors"
)

var (
	maxFilesToRead       = 5000
	errFileReadOverLimit = errors.New("read files over limit")
)

func DirectoryHash(srcPath string, excludePatterns, includeFiles []string) (string, error) {
	srcPath, err := filepath.Abs(srcPath)
	if err != nil {
		return "", err
	}

	// Stat dir / file
	hash := sha256.New()
	fileInfo, err := os.Stat(srcPath)
	if err != nil {
		return "", err
	}

	// Hash file
	if !fileInfo.IsDir() {
		return "", nil
	}

	// Fix the source path to work with long path names. This is a no-op
	// on platforms other than Windows.
	if runtime.GOOS == "windows" {
		srcPath = longpath.AddPrefix(srcPath)
	}

	pm, err := patternmatcher.New(excludePatterns)
	if err != nil {
		return "", err
	}

	// In general we log errors here but ignore them because
	// during e.g. a diff operation the container can continue
	// mutating the filesystem and we can see transient errors
	// from this
	stat, err := os.Lstat(srcPath)
	if err != nil {
		return "", err
	}

	if !stat.IsDir() {
		return "", errors.Errorf("Path %s is not a directory", srcPath)
	}

	include := "."
	seen := make(map[string]bool)

	retFiles := []string{}
	walkRoot := filepath.Join(srcPath, include)
	err = filepath.Walk(walkRoot, func(filePath string, f os.FileInfo, err error) error {
		if err != nil {
			return errors.Errorf("Hash: Can't stat file %s to hash: %s", srcPath, err)
		}

		if len(retFiles) >= maxFilesToRead {
			return errFileReadOverLimit
		}

		relFilePath, err := filepath.Rel(srcPath, filePath)
		if err != nil {
			// Error getting relative path OR we are looking
			// at the source directory path. Skip in both situations.
			return err
		}
		relFilePath = filepath.ToSlash(relFilePath)

		// Ensure file affects build context
		include := false
		for _, f := range includeFiles {
			if strings.HasPrefix(relFilePath, f) {
				include = true
			}
		}
		if !include {
			return nil
		}

		skip := false

		// If "include" is an exact match for the current file
		// then even if there's an "excludePatterns" pattern that
		// matches it, don't skip it. IOW, assume an explicit 'include'
		// is asking for that file no matter what - which is true
		// for some files, like .dockerignore and Dockerfile (sometimes)
		if relFilePath != "." {
			skip, err = pm.MatchesOrParentMatches(relFilePath)
			if err != nil {
				return errors.Errorf("Error matching %s: %v", relFilePath, err)
			}
		}

		if skip {
			// If we want to skip this file and its a directory
			// then we should first check to see if there's an
			// excludes pattern (e.g. !dir/file) that starts with this
			// dir. If so then we can't skip this dir.

			// Its not a dir then so we can just return/skip.
			if !f.IsDir() {
				return nil
			}

			// No exceptions (!...) in patterns so just skip dir
			if !pm.Exclusions() {
				return filepath.SkipDir
			}
			dirSlash := relFilePath + string(filepath.Separator)
			for _, pat := range pm.Patterns() {
				if !pat.Exclusion() {
					continue
				}

				if strings.HasPrefix(pat.String()+string(filepath.Separator), dirSlash) {
					// found a match - so can't skip this dir
					return nil
				}
			}

			// No matching exclusion dir so just skip dir
			return filepath.SkipDir
		}

		if seen[relFilePath] {
			return nil
		}

		// Path is enough
		seen[relFilePath] = true
		if !f.IsDir() {
			// Check file change
			checksum, err := hashFileCRC32(filePath, 0xedb88320)
			if err != nil {
				return nil
			}

			retFiles = append(retFiles, relFilePath+";"+checksum)
		}

		return nil
	})
	if err != nil && !errors.Is(err, errFileReadOverLimit) {
		return "", errors.Errorf("Error hashing %s: %v", srcPath, err)
	}

	// add to hash
	sort.Strings(retFiles)
	for _, f := range retFiles {
		_, _ = hash.Write([]byte(f))
	}
	if len(retFiles) == 0 {
		return "", nil
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func hashFileCRC32(filePath string, polynomial uint32) (string, error) {
	//Initialize an empty return string now in case an error has to be returned
	var returnCRC32String string

	//Open the fhe file located at the given path and check for errors
	file, err := os.Open(filePath)
	if err != nil {
		return returnCRC32String, err
	}

	//Tell the program to close the file when the function returns
	defer file.Close()

	//Create the table with the given polynomial
	tablePolynomial := crc32.MakeTable(polynomial)

	//Open a new hash interface to write the file to
	hash := crc32.New(tablePolynomial)

	//Copy the file in the interface
	if _, err := io.Copy(hash, file); err != nil {
		return returnCRC32String, err
	}

	//Generate the hash
	hashInBytes := hash.Sum(nil)[:]

	//Encode the hash to a string
	returnCRC32String = hex.EncodeToString(hashInBytes)

	//Return the output
	return returnCRC32String, nil
}
