package download

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/loft-sh/devpod/pkg/gitcredentials"
	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
)

func Head(rawURL string) (int, error) {
	req, err := http.NewRequest("HEAD", rawURL, nil)
	if err != nil {
		return 0, err
	}

	resp, err := devpodhttp.GetHTTPClient().Do(req)
	if err != nil {
		return 0, errors.Wrap(err, "download file")
	}

	return resp.StatusCode, nil
}

func File(rawURL string, log log.Logger) (io.ReadCloser, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}

	if parsed.Host == "github.com" {
		// check if we can access the url
		code, err := Head(rawURL)
		if err != nil {
			return nil, err
		} else if code == 404 {
			// check if github release
			path := parsed.Path
			org, repo, release, file := parseGithubURL(path)
			if org != "" {
				// try to download with credentials if its a release
				log.Debugf("Try to find credentials for github")
				credentials, err := gitcredentials.GetCredentials(&gitcredentials.GitCredentials{
					Protocol: parsed.Scheme,
					Host:     parsed.Host,
					Path:     parsed.Path,
				})
				if err == nil && credentials != nil && credentials.Password != "" {
					log.Debugf("Make request with credentials")
					return downloadGithubRelease(org, repo, release, file, credentials.Password)
				}
			}
		}
	}

	resp, err := devpodhttp.GetHTTPClient().Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "download file")
	} else if resp.StatusCode >= 400 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("received status code %d when trying to download %s", resp.StatusCode, rawURL)
	}

	return resp.Body, nil
}

type GithubRelease struct {
	Assets []GithubReleaseAsset `json:"assets,omitempty"`
}

type GithubReleaseAsset struct {
	ID   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

func downloadGithubRelease(org, repo, release, file, token string) (io.ReadCloser, error) {
	releaseURL := ""
	if release == "" {
		releaseURL = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", org, repo)
	} else {
		releaseURL = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", org, repo, release)
	}

	req, err := http.NewRequest("GET", releaseURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := devpodhttp.GetHTTPClient().Do(req)
	if err != nil {
		return nil, err
	} else if resp.StatusCode >= 400 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("received status code %d when trying to reach %s", resp.StatusCode, releaseURL)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	releaseObj := &GithubRelease{}
	err = json.Unmarshal(raw, releaseObj)
	if err != nil {
		return nil, err
	}

	var releaseAsset *GithubReleaseAsset
	for _, asset := range releaseObj.Assets {
		asset := asset
		if asset.Name == file {
			releaseAsset = &asset
			break
		}
	}
	if releaseAsset == nil {
		return nil, fmt.Errorf("couldn't find asset %s in github release (%s)", file, releaseURL)
	}

	req, err = http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/assets/%d", org, repo, releaseAsset.ID), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/octet-stream")
	downloadResp, err := devpodhttp.GetHTTPClient().Do(req)
	if err != nil {
		return nil, err
	} else if downloadResp.StatusCode >= 400 {
		_ = downloadResp.Body.Close()
		return nil, fmt.Errorf("received status code %d when trying to reach %s", downloadResp.StatusCode, releaseURL)
	}

	return downloadResp.Body, nil
}

func parseGithubURL(path string) (org, repo, release, file string) {
	splitted := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(splitted) != 6 {
		return "", "", "", ""
	} else if splitted[2] != "releases" {
		return "", "", "", ""
	} else if (splitted[3] != "latest" || splitted[4] != "download") && splitted[3] != "download" {
		return "", "", "", ""
	}

	if splitted[3] == "latest" {
		return splitted[0], splitted[1], "", splitted[5]
	}

	return splitted[0], splitted[1], splitted[4], splitted[5]
}
