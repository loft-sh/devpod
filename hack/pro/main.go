package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

//go:embed provider.yaml
var provider string

var checksumMap = map[string]string{
	"./release/devpod-linux-amd64":       "##CHECKSUM_LINUX_AMD64##",
	"./release/devpod-linux-arm64":       "##CHECKSUM_LINUX_ARM64##",
	"./release/devpod-darwin-amd64":      "##CHECKSUM_DARWIN_AMD64##",
	"./release/devpod-darwin-arm64":      "##CHECKSUM_DARWIN_ARM64##",
	"./release/devpod-windows-amd64.exe": "##CHECKSUM_WINDOWS_AMD64##",
}

func main() {
	partial := os.Getenv("PARTIAL") == "true"
	sourceFile, ok := os.LookupEnv("SOURCE_FILE")
	absPath := ""

	if ok {
		var err error

		absPath, err = filepath.Abs(sourceFile)
		if err != nil {
			panic(err)
		}

		providerBytes, err := os.ReadFile(absPath)
		if err != nil {
			panic(err)
		}

		provider = string(providerBytes)
	}

	replaced := strings.Replace(provider, "##VERSION##", os.Args[1], -1)
	for k, v := range checksumMap {
		checksum, err := File(k)
		if err != nil {
			if partial {
				continue
			}

			panic(fmt.Errorf("generate checksum for %s: %w", k, err))
		}

		replaced = strings.Replace(replaced, v, checksum, -1)
	}

	if ok {
		err := os.WriteFile(absPath, []byte(replaced), 0644)
		if err != nil {
			panic(err)
		}
	} else {
		fmt.Println(replaced)
	}
}

// File hashes a given file to a sha256 string
func File(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()

	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	return strings.ToLower(hex.EncodeToString(hash.Sum(nil))), nil
}
