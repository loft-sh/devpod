package binaries

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/copy"
	"github.com/loft-sh/devpod/pkg/download"
	"github.com/loft-sh/devpod/pkg/extract"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/hash"
	"github.com/pkg/errors"
)

func ToEnvironmentWithBinaries(context string, workspace *provider2.Workspace, machine *provider2.Machine, options map[string]config.OptionValue, config *provider2.ProviderConfig, extraEnv map[string]string, log log.Logger) ([]string, error) {
	environ := provider2.ToEnvironment(workspace, machine, options, extraEnv)
	binariesMap, err := GetBinaries(context, config)
	if err != nil {
		return nil, err
	}

	for k, v := range binariesMap {
		environ = append(environ, k+"="+v)
	}
	return environ, nil
}

func GetBinariesFrom(config *provider2.ProviderConfig, binariesDir string) (map[string]string, error) {
	retBinaries := map[string]string{}
	for binaryName, binaryLocations := range config.Binaries {
		for _, binary := range binaryLocations {
			if binary.OS != runtime.GOOS || binary.Arch != runtime.GOARCH {
				continue
			}

			// get binaries
			targetFolder := filepath.Join(binariesDir, strings.ToLower(binaryName))
			binaryPath := getBinaryPath(binary, targetFolder)
			_, err := os.Stat(binaryPath)
			if err != nil {
				return nil, fmt.Errorf("error trying to find binary %s: %w", binaryName, err)
			}

			retBinaries[binaryName] = binaryPath
		}
		if retBinaries[binaryName] == "" {
			return nil, fmt.Errorf("cannot find provider binary %s, because no binary location matched OS %s and ARCH %s", binaryName, runtime.GOOS, runtime.GOARCH)
		}
	}

	return retBinaries, nil
}

func GetBinaries(context string, config *provider2.ProviderConfig) (map[string]string, error) {
	binariesDir, err := provider2.GetProviderBinariesDir(context, config.Name)
	if err != nil {
		return nil, err
	}

	return GetBinariesFrom(config, binariesDir)
}

func DownloadBinaries(binaries map[string][]*provider2.ProviderBinary, targetFolder string, log log.Logger) (map[string]string, error) {
	retBinaries := map[string]string{}
	for binaryName, binaryLocations := range binaries {
		for _, binary := range binaryLocations {
			if binary.OS != runtime.GOOS || binary.Arch != runtime.GOARCH {
				continue
			}

			// check if binary is correct
			targetFolder := filepath.Join(targetFolder, strings.ToLower(binaryName))
			binaryPath := getBinaryPath(binary, targetFolder)
			if verifyBinary(binaryPath, binary.Checksum) || fromCache(binary, targetFolder, log) {
				retBinaries[binaryName] = binaryPath
				continue
			}

			// try to download the binary
			for i := 0; i < 3; i++ {
				binaryPath, err := downloadBinary(binaryName, binary, targetFolder, log)
				if err != nil {
					return nil, errors.Wrapf(err, "downloading binary %s", binaryName)
				}

				if binary.Checksum != "" {
					fileHash, err := hash.File(binaryPath)
					if err != nil {
						_ = os.Remove(binaryPath)
						log.Errorf("Error hashing %s: %v", binaryPath, err)
						continue
					} else if !strings.EqualFold(fileHash, binary.Checksum) {
						_ = os.Remove(binaryPath)
						log.Errorf("Unexpected file checksum %s != %s for binary %s", strings.ToLower(fileHash), strings.ToLower(binary.Checksum), binaryName)
						time.Sleep(250 * time.Millisecond)
						continue
					}
				}

				toCache(binary, binaryPath, log)
				retBinaries[binaryName] = binaryPath
				break
			}
			if retBinaries[binaryName] == "" {
				return nil, fmt.Errorf("cannot download provider binary %s, because checksum check has failed", binaryName)
			}
		}
		if retBinaries[binaryName] == "" {
			return nil, fmt.Errorf("cannot download provider binary %s, because no binary location matched OS %s and ARCH %s", binaryName, runtime.GOOS, runtime.GOARCH)
		}
	}

	return retBinaries, nil
}

func toCache(binary *provider2.ProviderBinary, binaryPath string, log log.Logger) {
	if !isRemotePath(binary.Path) {
		return
	}

	cachedBinaryPath := getCachedBinaryPath(binary.Path)
	err := os.MkdirAll(filepath.Dir(cachedBinaryPath), 0777)
	if err != nil {
		return
	}

	err = copy.File(binaryPath, cachedBinaryPath, 0755)
	if err != nil {
		log.Warnf("Error copying binary to cache: %v", err)
		return
	}
}

func fromCache(binary *provider2.ProviderBinary, targetFolder string, log log.Logger) bool {
	if !isRemotePath(binary.Path) {
		return false
	}

	binaryPath := getBinaryPath(binary, targetFolder)
	cachedBinaryPath := getCachedBinaryPath(binary.Path)
	if !verifyBinary(cachedBinaryPath, binary.Checksum) {
		return false
	}

	err := os.MkdirAll(path.Dir(binaryPath), 0755)
	if err != nil {
		log.Warnf("Error creating directory %s: %v", path.Dir(binaryPath), err)
		return false
	}

	err = copy.File(cachedBinaryPath, binaryPath, 0755)
	if err != nil {
		log.Warnf("Error copying cached binary from %s to %s: %v", cachedBinaryPath, binaryPath, err)
		return false
	}

	err = os.Chmod(binaryPath, 0755)
	if err != nil {
		log.Warnf("Error chmod binary %s: %v", binaryPath, err)
		return false
	}

	return true
}

func getCachedBinaryPath(url string) string {
	return filepath.Join(os.TempDir(), "devpod-binaries", hash.String(url)[:16])
}

func verifyBinary(binaryPath, checksum string) bool {
	_, err := os.Stat(binaryPath)
	if err != nil {
		return false
	}

	// verify checksum
	if checksum != "" {
		fileHash, err := hash.File(binaryPath)
		if err != nil || !strings.EqualFold(fileHash, checksum) {
			_ = os.Remove(binaryPath)
			return false
		}
	}

	return true
}

func getBinaryPath(binary *provider2.ProviderBinary, targetFolder string) string {
	if filepath.IsAbs(binary.Path) {
		return binary.Path
	}

	// check if download
	if !isRemotePath(binary.Path) {
		return localTargetPath(binary, targetFolder)
	}

	// check if archive
	if binary.ArchivePath != "" {
		return path.Join(filepath.ToSlash(targetFolder), binary.ArchivePath)
	}

	// determine binary name
	name := binary.Name
	if name == "" {
		name = path.Base(binary.Path)
		if runtime.GOOS == "windows" && !strings.HasSuffix(name, ".exe") {
			name += ".exe"
		}
	}

	return path.Join(filepath.ToSlash(targetFolder), name)
}

func isRemotePath(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

func downloadBinary(binaryName string, binary *provider2.ProviderBinary, targetFolder string, log log.Logger) (string, error) {
	// check if local
	_, err := os.Stat(binary.Path)
	if err == nil {
		if filepath.IsAbs(binary.Path) {
			return binary.Path, nil
		}

		err := os.MkdirAll(targetFolder, 0755)
		if err != nil {
			return "", errors.Wrap(err, "create folder")
		}

		targetPath := localTargetPath(binary, targetFolder)
		err = copyLocal(binary, targetFolder)
		if err != nil {
			_ = os.RemoveAll(targetFolder)
			return "", err
		}

		return targetPath, nil
	}

	// check if download
	if !strings.HasPrefix(binary.Path, "http://") && !strings.HasPrefix(binary.Path, "https://") {
		// check if local already copied
		targetPath := localTargetPath(binary, targetFolder)
		_, err := os.Stat(targetPath)
		if err == nil {
			return targetPath, nil
		}

		return "", fmt.Errorf("cannot download %s as scheme is missing", binary.Path)
	}

	// create target folder
	err = os.MkdirAll(targetFolder, 0755)
	if err != nil {
		return "", errors.Wrap(err, "create folder")
	}

	// check if archive
	if binary.ArchivePath != "" {
		targetPath, err := downloadArchive(binaryName, binary, targetFolder, log)
		if err != nil {
			_ = os.RemoveAll(targetFolder)
			return "", err
		}

		err = os.Chmod(targetPath, 0755)
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

	err = os.Chmod(targetPath, 0755)
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
	body, err := download.File(binary.Path, log)
	if err != nil {
		return "", errors.Wrap(err, "download binary")
	}
	defer body.Close()

	file, err := os.Create(targetPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, body)
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
	body, err := download.File(binary.Path, log)
	if err != nil {
		return "", err
	}
	defer body.Close()

	// determine archive
	if strings.HasSuffix(binary.Path, ".gz") || strings.HasSuffix(binary.Path, ".tar") || strings.HasSuffix(binary.Path, ".tgz") {
		err = extract.Extract(body, targetFolder)
		if err != nil {
			return "", err
		}

		return targetPath, nil
	} else if strings.HasSuffix(binary.Path, ".zip") {
		tempFile, err := downloadToTempFile(body)
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

func localTargetPath(binary *provider2.ProviderBinary, targetFolder string) string {
	name := binary.Name
	if name == "" {
		name = path.Base(binary.Path)
	}

	targetPath := filepath.Join(targetFolder, name)
	return targetPath
}

func copyLocal(binary *provider2.ProviderBinary, targetPath string) error {
	// determine binary name
	targetPathStat, err := os.Stat(targetPath)
	if err == nil {
		binaryStat, err := os.Stat(binary.Path)
		if err != nil {
			return err
		} else if targetPathStat.Size() == binaryStat.Size() {
			return nil
		}
	}

	err = copy.File(binary.Path, targetPath, 0755)
	if err != nil {
		return err
	}

	return nil
}
