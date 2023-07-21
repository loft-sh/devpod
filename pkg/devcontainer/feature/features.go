package feature

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/extract"
	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/hash"
	"github.com/pkg/errors"
)

const DEVCONTAINER_MANIFEST_MEDIATYPE = "application/vnd.devcontainers"

var directTarballRegEx = regexp.MustCompile("devcontainer-feature-([a-zA-Z0-9_-]+).tgz")

func getFeatureInstallWrapperScript(idWithoutVersion string, feature *config.FeatureConfig, options []string) string {
	id := escapeQuotesForShell(idWithoutVersion)
	name := escapeQuotesForShell(feature.Name)
	description := escapeQuotesForShell(feature.Description)
	version := escapeQuotesForShell(feature.Version)
	documentation := escapeQuotesForShell(feature.DocumentationURL)
	optionsIndented := escapeQuotesForShell("    " + strings.Join(options, "\n    "))

	warningHeader := ""
	if feature.Deprecated {
		warningHeader += `(!) WARNING: Using the deprecated Feature "${escapeQuotesForShell(feature.id)}". This Feature will no longer receive any further updates/support.\n`
	}

	echoWarning := ""
	if warningHeader != "" {
		echoWarning = `echo '` + warningHeader + `'`
	}

	errorMessage := `ERROR: Feature "` + name + `" (` + id + `) failed to install!`
	troubleshootingMessage := ""
	if documentation != "" {
		troubleshootingMessage = ` Look at the documentation at ${documentation} for help troubleshooting this error.`
	}

	return `#!/bin/sh
set -e

on_exit () {
	[ $? -eq 0 ] && exit
	echo '` + errorMessage + troubleshootingMessage + `'
}

trap on_exit EXIT

echo ===========================================================================
` + echoWarning + `
echo 'Feature       : ` + name + `'
echo 'Description   : ` + description + `'
echo 'Id            : ` + id + `'
echo 'Version       : ` + version + `'
echo 'Documentation : ` + documentation + `'
echo 'Options       :'
echo '` + optionsIndented + `'
echo ===========================================================================

set -a
. ../devcontainer-features.builtin.env
. ./devcontainer-features.env
set +a

chmod +x ./install.sh
./install.sh
`
}

func escapeQuotesForShell(str string) string {
	// The `input` is expected to be a string which will be printed inside single quotes
	// by the caller. This means we need to escape any nested single quotes within the string.
	// We can do this by ending the first string with a single quote ('), printing an escaped
	// single quote (\'), and then opening a new string (').
	return strings.ReplaceAll(str, "'", `'\''`)
}

func ProcessFeatureID(id, configDir string, log log.Logger) (string, error) {
	if strings.HasPrefix(id, "https://") || strings.HasPrefix(id, "http://") {
		log.Debugf("Process url feature")
		return processDirectTarFeature(id, log)
	} else if strings.HasPrefix(id, "./") || strings.HasPrefix(id, "../") {
		log.Debugf("Process local feature")
		return filepath.Abs(path.Join(filepath.ToSlash(configDir), id))
	}

	// get oci feature
	log.Debugf("Process OCI feature")
	return processOCIFeature(id, log)
}

func processOCIFeature(id string, log log.Logger) (string, error) {
	// feature already exists?
	featureFolder := getFeaturesTempFolder(id)
	featureExtractedFolder := filepath.Join(featureFolder, "extracted")
	_, err := os.Stat(featureExtractedFolder)
	if err == nil {
		// make sure feature.json is there as well
		_, err = os.Stat(filepath.Join(featureExtractedFolder, config.DEVCONTAINER_FEATURE_FILE_NAME))
		if err == nil {
			return featureExtractedFolder, nil
		} else {
			log.Debugf("Feature folder already exists but seems to be empty")
			_ = os.RemoveAll(featureFolder)
		}
	}

	ref, err := name.ParseReference(id)
	if err != nil {
		return "", err
	}

	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return "", err
	}

	destFile := filepath.Join(featureFolder, "feature.tgz")
	err = downloadLayer(img, id, destFile, log)
	if err != nil {
		return "", err
	}

	file, err := os.Open(destFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	log.Debugf("Extract feature into %s", featureExtractedFolder)
	err = extract.Extract(file, featureExtractedFolder)
	if err != nil {
		_ = os.RemoveAll(featureExtractedFolder)
		return "", err
	}

	return featureExtractedFolder, nil
}

func downloadLayer(img v1.Image, id, destFile string, log log.Logger) error {
	manifest, err := img.Manifest()
	if err != nil {
		return err
	} else if manifest.Config.MediaType != DEVCONTAINER_MANIFEST_MEDIATYPE {
		return fmt.Errorf("incorrect manifest type %s, expected %s", manifest.Config.MediaType, DEVCONTAINER_MANIFEST_MEDIATYPE)
	} else if len(manifest.Layers) == 0 {
		return fmt.Errorf("unexpected amount of layers, expected at least 1")
	}

	// download layer
	log.Debugf("Download feature %s layer %s into %s...", id, manifest.Layers[0].Digest, destFile)
	layer, err := img.LayerByDigest(manifest.Layers[0].Digest)
	if err != nil {
		return errors.Wrap(err, "retrieve layer")
	}

	data, err := layer.Uncompressed()
	if err != nil {
		return errors.Wrap(err, "download")
	}
	defer data.Close()

	err = os.MkdirAll(filepath.Dir(destFile), 0755)
	if err != nil {
		return errors.Wrap(err, "create target folder")
	}

	file, err := os.Create(destFile)
	if err != nil {
		return errors.Wrap(err, "create file")
	}
	defer file.Close()

	_, err = io.Copy(file, data)
	if err != nil {
		return errors.Wrap(err, "download layer")
	}

	return nil
}

func processDirectTarFeature(id string, log log.Logger) (string, error) {
	downloadBase := id[strings.LastIndex(id, "/"):]
	if !directTarballRegEx.MatchString(downloadBase) {
		return "", fmt.Errorf("expected tarball name to follow 'devcontainer-feature-<feature-id>.tgz' format.  Received '%s' ", downloadBase)
	}

	// feature already exists?
	featureFolder := getFeaturesTempFolder(id)
	featureExtractedFolder := filepath.Join(featureFolder, "extracted")
	_, err := os.Stat(featureExtractedFolder)
	if err == nil {
		return featureExtractedFolder, nil
	}

	// download feature tarball
	downloadFile := filepath.Join(featureFolder, "feature.tgz")
	err = downloadFeatureFromURL(id, downloadFile, log)
	if err != nil {
		return "", err
	}

	// extract file
	file, err := os.Open(downloadFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// extract tar.gz
	err = extract.Extract(file, featureExtractedFolder)
	if err != nil {
		_ = os.RemoveAll(featureExtractedFolder)
		return "", errors.Wrap(err, "extract folder")
	}

	return featureExtractedFolder, nil
}

func downloadFeatureFromURL(url string, destFile string, log log.Logger) error {
	// create the features temp folder
	err := os.MkdirAll(filepath.Dir(destFile), 0755)
	if err != nil {
		return errors.Wrap(err, "create feature folder")
	}

	// initiate download
	log.Debugf("Download feature from %s", url)
	resp, err := devpodhttp.GetHTTPClient().Get(url)
	if err != nil {
		return errors.Wrap(err, "make request")
	}
	defer resp.Body.Close()

	// download the tar.gz file
	file, err := os.Create(destFile)
	if err != nil {
		return errors.Wrap(err, "create download file")
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return errors.Wrap(err, "download feature")
	}

	return nil
}

func getFeaturesTempFolder(id string) string {
	hashedID := hash.String(id)[:10]
	return filepath.Join(os.TempDir(), "devpod", "features", hashedID)
}
