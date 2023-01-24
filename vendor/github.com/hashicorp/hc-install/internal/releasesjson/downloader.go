package releasesjson

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	"github.com/hashicorp/hc-install/internal/httpclient"
)

type Downloader struct {
	Logger           *log.Logger
	VerifyChecksum   bool
	ArmoredPublicKey string
	BaseURL          string
}

func (d *Downloader) DownloadAndUnpack(ctx context.Context, pv *ProductVersion, dstDir string) error {
	if len(pv.Builds) == 0 {
		return fmt.Errorf("no builds found for %s %s", pv.Name, pv.Version)
	}

	pb, ok := pv.Builds.FilterBuild(runtime.GOOS, runtime.GOARCH, "zip")
	if !ok {
		return fmt.Errorf("no ZIP archive found for %s %s %s/%s",
			pv.Name, pv.Version, runtime.GOOS, runtime.GOARCH)
	}

	var verifiedChecksum HashSum
	if d.VerifyChecksum {
		v := &ChecksumDownloader{
			BaseURL:          d.BaseURL,
			ProductVersion:   pv,
			Logger:           d.Logger,
			ArmoredPublicKey: d.ArmoredPublicKey,
		}
		verifiedChecksums, err := v.DownloadAndVerifyChecksums()
		if err != nil {
			return err
		}
		var ok bool
		verifiedChecksum, ok = verifiedChecksums[pb.Filename]
		if !ok {
			return fmt.Errorf("no checksum found for %q", pb.Filename)
		}
	}

	client := httpclient.NewHTTPClient()

	archiveURL := pb.URL
	if d.BaseURL != "" {
		// ensure that absolute download links from mocked responses
		// are still pointing to the mock server if one is set
		baseURL, err := url.Parse(d.BaseURL)
		if err != nil {
			return err
		}

		u, err := url.Parse(archiveURL)
		if err != nil {
			return err
		}
		u.Scheme = baseURL.Scheme
		u.Host = baseURL.Host
		archiveURL = u.String()
	}

	d.Logger.Printf("downloading archive from %s", archiveURL)
	resp, err := client.Get(archiveURL)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to download ZIP archive from %q: %s", archiveURL, resp.Status)
	}

	defer resp.Body.Close()

	var pkgReader io.Reader
	pkgReader = resp.Body

	contentType := resp.Header.Get("content-type")
	if !contentTypeIsZip(contentType) {
		return fmt.Errorf("unexpected content-type: %s (expected any of %q)",
			contentType, zipMimeTypes)
	}

	expectedSize := resp.ContentLength

	if d.VerifyChecksum {
		d.Logger.Printf("verifying checksum of %q", pb.Filename)
		// provide extra reader to calculate & compare checksum
		var buf bytes.Buffer
		r := io.TeeReader(resp.Body, &buf)
		pkgReader = &buf

		err := compareChecksum(d.Logger, r, verifiedChecksum, pb.Filename, expectedSize)
		if err != nil {
			return err
		}
	}

	pkgFile, err := ioutil.TempFile("", pb.Filename)
	if err != nil {
		return err
	}
	defer pkgFile.Close()

	d.Logger.Printf("copying %q (%d bytes) to %s", pb.Filename, expectedSize, pkgFile.Name())
	// Unless the bytes were already downloaded above for checksum verification
	// this may take a while depending on network connection as the io.Reader
	// is expected to be http.Response.Body which streams the bytes
	// on demand over the network.
	bytesCopied, err := io.Copy(pkgFile, pkgReader)
	if err != nil {
		return err
	}
	d.Logger.Printf("copied %d bytes to %s", bytesCopied, pkgFile.Name())

	if expectedSize != 0 && bytesCopied != int64(expectedSize) {
		return fmt.Errorf("unexpected size (downloaded: %d, expected: %d)",
			bytesCopied, expectedSize)
	}

	r, err := zip.OpenReader(pkgFile.Name())
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		srcFile, err := f.Open()
		if err != nil {
			return err
		}

		d.Logger.Printf("unpacking %s to %s", f.Name, dstDir)
		dstPath := filepath.Join(dstDir, f.Name)
		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}

		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			return err
		}
		srcFile.Close()
		dstFile.Close()
	}

	return nil
}

// The production release site uses consistent single mime type
// but mime types are platform-dependent
// and we may use different OS under test
var zipMimeTypes = []string{
	"application/x-zip-compressed", // Windows
	"application/zip",              // Unix
}

func contentTypeIsZip(contentType string) bool {
	for _, mt := range zipMimeTypes {
		if mt == contentType {
			return true
		}
	}
	return false
}
