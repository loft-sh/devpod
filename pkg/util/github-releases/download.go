package githubreleases

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/go-github/github"
)

func DownloadLatestBuildxRelease(owner, repoName string) (string, error) {
	// Initialize the GitHub client
	client := github.NewClient(nil)

	destination := ""
	if _, err := os.Stat("/usr/libexec/docker/cli-plugins"); err == nil {
		destination = "/usr/libexec/docker/cli-plugins"
	} else if _, err := os.Stat("/usr/lib/docker/cli-plugins"); err == nil {
		destination = "/usr/lib/docker/cli-plugins"
	}
	destination = filepath.Join(destination, "docker-buildx")

	// Get the latest release
	release, _, err := client.Repositories.GetLatestRelease(context.Background(), owner, repoName)
	if err != nil {
		return "", fmt.Errorf("error getting latest release: %s", err)
	}

	for _, asset := range release.Assets {
		if strings.Contains(*asset.Name, "linux-"+runtime.GOARCH) && !strings.HasSuffix(*asset.Name, ".json") {
			response, err := http.Get(asset.GetBrowserDownloadURL())
			if err != nil {
				return "", err
			}
			defer response.Body.Close()

			file, err := os.CreateTemp("/tmp/", "*.tmp")
			if err != nil {
				return "", err
			}
			defer file.Close()

			err = file.Chmod(0o755)
			if err != nil {
				return "", err
			}

			_, err = io.Copy(file, response.Body)
			if err != nil {
				return "", err
			}

			err = os.Rename(file.Name(), destination)
			if err != nil {
				return "", err
			}
		}
	}

	return destination, nil
}
