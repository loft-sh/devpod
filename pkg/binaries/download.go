package binaries

import (
	"crypto/tls"
	"fmt"
	"github.com/loft-sh/devpod/pkg/copy"
	"github.com/loft-sh/devpod/pkg/extract"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

func DownloadBinaries(binaries map[string][]*provider2.ProviderBinary, targetFolder string, log log.Logger) (map[string]string, error) {
	retBinaries := map[string]string{}
	for binaryName, binaryLocations := range binaries {
		for _, binary := range binaryLocations {
			if binary.OS != runtime.GOOS && binary.Arch != runtime.GOARCH {
				continue
			}

			binaryPath, err := downloadBinary(binaryName, binary, filepath.Join(targetFolder, strings.ToLower(binaryName)), log)
			if err != nil {
				return nil, errors.Wrapf(err, "downloading binary %s", binaryName)
			}

			retBinaries[binaryName] = binaryPath
		}
		if retBinaries[binaryName] == "" {
			log.Infof("Skip downloading binary %s, because no binary location matched OS %s and ARCH %s", runtime.GOOS, runtime.GOARCH)
		}
	}

	return retBinaries, nil
}

func downloadBinary(binaryName string, binary *provider2.ProviderBinary, targetFolder string, log log.Logger) (string, error) {
	err := os.MkdirAll(targetFolder, os.ModePerm)
	if err != nil {
		return "", errors.Wrap(err, "create folder")
	}

	// check if local
	_, err = os.Stat(binary.Path)
	if err != nil {
		if filepath.IsAbs(binary.Path) {
			return binary.Path, nil
		}

		targetPath, err := copyLocal(binary, targetFolder)
		if err != nil {
			_ = os.RemoveAll(targetFolder)
			return "", err
		}

		return targetPath, nil
	}

	// check if download
	if !strings.HasPrefix(binary.Path, "http://") && !strings.HasPrefix(binary.Path, "https://") {
		return "", fmt.Errorf("cannot download %s as scheme is missing", binary.Path)
	}

	// check if archive
	if binary.ArchivePath != "" {
		targetPath, err := downloadArchive(binaryName, binary, targetFolder, log)
		if err != nil {
			_ = os.RemoveAll(targetFolder)
			return "", err
		}

		err = os.Chmod(targetPath, os.ModePerm)
		if err != nil {
			return "", err
		}

		return targetPath, nil
	}

	// download file
	targetPath, err := downloadFile(binaryName, binary, targetFolder, log)
	if err != nil {
		_ = os.RemoveAll(targetFolder)
		return "", err
	}

	err = os.Chmod(targetPath, os.ModePerm)
	if err != nil {
		return "", err
	}

	return targetPath, nil
}

func downloadFile(binaryName string, binary *provider2.ProviderBinary, targetFolder string, log log.Logger) (string, error) {
	// determine binary name
	name := binary.Name
	if name == "" {
		name = path.Base(binary.Path)
		if runtime.GOOS == "windows" && !strings.HasSuffix(name, ".exe") {
			name += ".exe"
		}
	}

	targetPath := path.Join(filepath.ToSlash(targetFolder), name)
	_, err := os.Stat(targetPath)
	if err == nil {
		return targetPath, nil
	}

	// initiate download
	log.Infof("Download binary %s from %s", binaryName, binary.Path)
	defer log.Debugf("Successfully downloaded binary %s", binary.Path)
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := httpClient.Get(binary.Path)
	if err != nil {
		return "", errors.Wrap(err, "download binary")
	}
	defer resp.Body.Close()

	file, err := os.Create(targetPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "download file")
	}

	return targetPath, nil
}

func downloadArchive(binaryName string, binary *provider2.ProviderBinary, targetFolder string, log log.Logger) (string, error) {
	targetPath := path.Join(filepath.ToSlash(targetFolder), binary.ArchivePath)
	_, err := os.Stat(targetPath)
	if err == nil {
		return targetPath, nil
	}

	// initiate download
	log.Infof("Download binary %s from %s", binaryName, binary.Path)
	defer log.Debugf("Successfully extracted & downloaded archive")
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := httpClient.Get(binary.Path)
	if err != nil {
		return "", errors.Wrap(err, "download binary")
	}
	defer resp.Body.Close()

	// determine archive
	if strings.HasSuffix(binary.Path, ".gz") || strings.HasSuffix(binary.Path, ".tar") || strings.HasSuffix(binary.Path, ".tgz") {
		err = extract.Extract(resp.Body, targetFolder)
		if err != nil {
			return "", err
		}

		return targetPath, nil
	} else if strings.HasSuffix(binary.Path, ".zip") {
		tempFile, err := downloadToTempFile(resp.Body)
		if err != nil {
			return "", err
		}
		defer os.Remove(tempFile)

		err = extract.UnzipFolder(tempFile, targetFolder)
		if err != nil {
			return "", err
		}

		return targetPath, nil
	}

	return "", fmt.Errorf("unrecognized archive format %s", binary.Path)
}

func downloadToTempFile(reader io.Reader) (string, error) {
	tempFile, err := os.CreateTemp("", "")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, reader)
	if err != nil {
		_ = os.Remove(tempFile.Name())
		return "", err
	}

	return tempFile.Name(), nil
}

func copyLocal(binary *provider2.ProviderBinary, targetFolder string) (string, error) {
	// determine binary name
	name := binary.Name
	if name == "" {
		name = path.Base(binary.Path)
	}

	targetPath := filepath.Join(targetFolder, name)
	_, err := os.Stat(targetPath)
	if err == nil {
		return targetPath, nil
	}

	err = copy.File(binary.Path, targetPath)
	if err != nil {
		return "", err
	}

	err = os.Chmod(targetPath, 0755)
	if err != nil {
		return "", err
	}

	return targetPath, nil
}
