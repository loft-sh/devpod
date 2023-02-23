package goland

import (
	"crypto/tls"
	"github.com/loft-sh/devpod/pkg/command"
	copy2 "github.com/loft-sh/devpod/pkg/copy"
	"github.com/loft-sh/devpod/pkg/extract"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"
)

const GolandDownloadAmd64 = "https://download.jetbrains.com/go/goland-2022.3.2.tar.gz"
const GolandDownloadArm64 = "https://download.jetbrains.com/go/goland-2022.3.2-aarch64.tar.gz"

const GolandArchive = "goland.tar.gz"

const GolandFolder = "/var/devpod"

func NewGolandServer(userName string, log log.Logger) ide.IDE {
	return &golandServer{
		userName: userName,
		log:      log,
	}
}

type golandServer struct {
	userName string
	log      log.Logger
}

func (o *golandServer) Install() error {
	baseFolder, err := getBaseFolder(o.userName)
	if err != nil {
		return err
	}
	targetLocation := GetGolandDirectory(baseFolder)

	_, err = os.Stat(targetLocation)
	if err == nil {
		o.log.Debugf("Goland already installed skip install")
		return nil
	}

	o.log.Debugf("Download goland archive")
	archivePath, err := downloadGoland(GolandFolder, o.log)
	if err != nil {
		return err
	}

	o.log.Infof("Extract goland...")
	err = extractGoland(archivePath, targetLocation)
	if err != nil {
		return err
	}

	err = copy2.ChownR(path.Join(baseFolder, ".cache"), o.userName)
	if err != nil {
		return errors.Wrap(err, "chown")
	}
	o.log.Infof("Successfully installed goland backend")
	return nil
}

func getBaseFolder(userName string) (string, error) {
	var err error
	homeFolder := ""
	if userName != "" {
		homeFolder, err = command.GetHome(userName)
	} else {
		homeFolder, err = homedir.Dir()
	}
	if err != nil {
		return "", err
	}

	return homeFolder, nil
}

func GetGolandDirectory(baseFolder string) string {
	return path.Join(baseFolder, ".cache", "JetBrains", "RemoteDev", "dist", "goland")
}

func extractGoland(fromPath string, toPath string) error {
	file, err := os.Open(fromPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return extract.Extract(file, toPath, extract.StripLevels(1))
}

func downloadGoland(targetFolder string, log log.Logger) (string, error) {
	err := os.MkdirAll(targetFolder, os.ModePerm)
	if err != nil {
		return "", err
	}

	downloadUrl := GolandDownloadAmd64
	if runtime.GOARCH == "arm64" {
		downloadUrl = GolandDownloadArm64
	}

	targetPath := path.Join(filepath.ToSlash(targetFolder), GolandArchive)
	_, err = os.Stat(targetPath)
	if err == nil {
		return targetPath, nil
	}

	// initiate download
	log.Infof("Download Goland from %s", downloadUrl)
	defer log.Debugf("Successfully downloaded Goland")
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := httpClient.Get(downloadUrl)
	if err != nil {
		return "", errors.Wrap(err, "download binary")
	}
	defer resp.Body.Close()

	file, err := os.Create(targetPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, &progressReader{
		reader:    resp.Body,
		totalSize: resp.ContentLength,
		log:       log,
	})
	if err != nil {
		return "", errors.Wrap(err, "download file")
	}

	return targetPath, nil
}

type progressReader struct {
	reader io.Reader

	lastMessage time.Time
	bytesRead   int64
	totalSize   int64
	log         log.Logger
}

func (p *progressReader) Read(b []byte) (n int, err error) {
	n, err = p.reader.Read(b)
	p.bytesRead += int64(n)
	if time.Since(p.lastMessage) > time.Second*1 {
		p.log.Infof("Downloaded %.2f / %.2f MB", float64(p.bytesRead)/1024/1024, float64(p.totalSize/1024/1024))
		p.lastMessage = time.Now()
	}

	return n, err
}
