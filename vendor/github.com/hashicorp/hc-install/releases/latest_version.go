package releases

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/internal/pubkey"
	rjson "github.com/hashicorp/hc-install/internal/releasesjson"
	isrc "github.com/hashicorp/hc-install/internal/src"
	"github.com/hashicorp/hc-install/internal/validators"
	"github.com/hashicorp/hc-install/product"
)

type LatestVersion struct {
	Product            product.Product
	Constraints        version.Constraints
	InstallDir         string
	Timeout            time.Duration
	IncludePrereleases bool

	SkipChecksumVerification bool

	// ArmoredPublicKey is a public PGP key in ASCII/armor format to use
	// instead of built-in pubkey to verify signature of downloaded checksums
	ArmoredPublicKey string

	apiBaseURL    string
	logger        *log.Logger
	pathsToRemove []string
}

func (*LatestVersion) IsSourceImpl() isrc.InstallSrcSigil {
	return isrc.InstallSrcSigil{}
}

func (lv *LatestVersion) SetLogger(logger *log.Logger) {
	lv.logger = logger
}

func (lv *LatestVersion) log() *log.Logger {
	if lv.logger == nil {
		return discardLogger
	}
	return lv.logger
}

func (lv *LatestVersion) Validate() error {
	if !validators.IsProductNameValid(lv.Product.Name) {
		return fmt.Errorf("invalid product name: %q", lv.Product.Name)
	}

	if !validators.IsBinaryNameValid(lv.Product.BinaryName()) {
		return fmt.Errorf("invalid binary name: %q", lv.Product.BinaryName())
	}

	return nil
}

func (lv *LatestVersion) Install(ctx context.Context) (string, error) {
	timeout := defaultInstallTimeout
	if lv.Timeout > 0 {
		timeout = lv.Timeout
	}
	ctx, cancelFunc := context.WithTimeout(ctx, timeout)
	defer cancelFunc()

	if lv.pathsToRemove == nil {
		lv.pathsToRemove = make([]string, 0)
	}

	dstDir := lv.InstallDir
	if dstDir == "" {
		var err error
		dirName := fmt.Sprintf("%s_*", lv.Product.Name)
		dstDir, err = ioutil.TempDir("", dirName)
		if err != nil {
			return "", err
		}
		lv.pathsToRemove = append(lv.pathsToRemove, dstDir)
		lv.log().Printf("created new temp dir at %s", dstDir)
	}
	lv.log().Printf("will install into dir at %s", dstDir)

	rels := rjson.NewReleases()
	if lv.apiBaseURL != "" {
		rels.BaseURL = lv.apiBaseURL
	}
	rels.SetLogger(lv.log())
	versions, err := rels.ListProductVersions(ctx, lv.Product.Name)
	if err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no versions found for %q", lv.Product.Name)
	}

	versionToInstall, ok := lv.findLatestMatchingVersion(versions, lv.Constraints)
	if !ok {
		return "", fmt.Errorf("no matching version found for %q", lv.Constraints)
	}

	d := &rjson.Downloader{
		Logger:           lv.log(),
		VerifyChecksum:   !lv.SkipChecksumVerification,
		ArmoredPublicKey: pubkey.DefaultPublicKey,
		BaseURL:          rels.BaseURL,
	}
	if lv.ArmoredPublicKey != "" {
		d.ArmoredPublicKey = lv.ArmoredPublicKey
	}
	if lv.apiBaseURL != "" {
		d.BaseURL = lv.apiBaseURL
	}
	err = d.DownloadAndUnpack(ctx, versionToInstall, dstDir)
	if err != nil {
		return "", err
	}

	execPath := filepath.Join(dstDir, lv.Product.BinaryName())

	lv.pathsToRemove = append(lv.pathsToRemove, execPath)

	lv.log().Printf("changing perms of %s", execPath)
	err = os.Chmod(execPath, 0o700)
	if err != nil {
		return "", err
	}

	return execPath, nil
}

func (lv *LatestVersion) Remove(ctx context.Context) error {
	if lv.pathsToRemove != nil {
		for _, path := range lv.pathsToRemove {
			err := os.RemoveAll(path)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (lv *LatestVersion) findLatestMatchingVersion(pvs rjson.ProductVersionsMap, vc version.Constraints) (*rjson.ProductVersion, bool) {
	versions := make(version.Collection, 0)
	for _, pv := range pvs.AsSlice() {
		if !lv.IncludePrereleases && pv.Version.Prerelease() != "" {
			// skip prereleases if desired
			continue
		}

		versions = append(versions, pv.Version)
	}

	if len(versions) == 0 {
		return nil, false
	}

	sort.Stable(versions)
	latestVersion := versions[len(versions)-1]

	return pvs[latestVersion.Original()], true
}
