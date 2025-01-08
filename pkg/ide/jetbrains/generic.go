package jetbrains

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/config"
	copy2 "github.com/loft-sh/devpod/pkg/copy"
	"github.com/loft-sh/devpod/pkg/extract"
	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/devpod/pkg/util"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
)

const (
	VersionOption       = "VERSION"
	DownloadAmd64Option = "DOWNLOAD_AMD64"
	DownloadArm64Option = "DOWNLOAD_ARM64"
)

func getLatestDownloadURL(code string, platform string) string {
	return fmt.Sprintf("https://download.jetbrains.com/product?code=%s&platform=%s", code, platform)
}

func getDownloadURLs(options ide.Options, values map[string]config.OptionValue, productCode string, templateAmd64 string, templateArm64 string) (string, string) {
	version := options.GetValue(values, VersionOption)
	var amd64Download, arm64Download string
	if version == "latest" {
		amd64Download = getLatestDownloadURL(productCode, "linux")
		arm64Download = getLatestDownloadURL(productCode, "linuxARM64")
	} else {
		amd64Download = options.GetValue(values, DownloadAmd64Option)
		if amd64Download == "" {
			amd64Download = fmt.Sprintf(templateAmd64, version)
		}
		arm64Download = options.GetValue(values, DownloadArm64Option)
		if arm64Download == "" {
			arm64Download = fmt.Sprintf(templateArm64, version)
		}
	}

	return amd64Download, arm64Download
}

type GenericOptions struct {
	ID          string
	DisplayName string

	DownloadAmd64 string
	DownloadArm64 string
}

func newGenericServer(userName string, options *GenericOptions, log log.Logger) *GenericJetBrainsServer {
	return &GenericJetBrainsServer{
		userName: userName,
		options:  options,
		log:      log,
	}
}

type GenericJetBrainsServer struct {
	userName string
	options  *GenericOptions
	log      log.Logger
}

func (o *GenericJetBrainsServer) OpenGateway(workspaceFolder, workspaceID string) error {
	o.log.Infof("Starting %s through JetBrains Gateway...", o.options.DisplayName)
	err := open.Run(`jetbrains-gateway://connect#idePath=` + url.QueryEscape(o.getDirectory(path.Join("/", "home", o.userName))) + `&projectPath=` + url.QueryEscape(workspaceFolder) + `&host=` + workspaceID + `.devpod&port=22&user=` + url.QueryEscape(o.userName) + `&type=ssh&deploy=false`)
	if err != nil {
		o.log.Debugf("Error opening jetbrains-gateway: %v", err)
		o.log.Errorf("Seems like you don't have JetBrains Gateway installed on your computer. Please install JetBrains Gateway via https://www.jetbrains.com/remote-development/gateway/")
		return err
	}
	return nil
}

func (o *GenericJetBrainsServer) GetVolume() string {
	return fmt.Sprintf("type=volume,src=devpod-%s,dst=%s", o.options.ID, o.getDownloadFolder())
}

func (o *GenericJetBrainsServer) getDownloadFolder() string {
	return fmt.Sprintf("/var/devpod/%s", o.options.ID)
}

func (o *GenericJetBrainsServer) Install() error {
	o.log.Debugf("Setup %s...", o.options.DisplayName)
	baseFolder, err := getBaseFolder(o.userName)
	if err != nil {
		return err
	}
	targetLocation := o.getDirectory(baseFolder)

	_, err = os.Stat(targetLocation)
	if err == nil {
		o.log.Debugf("Goland already installed skip install")
		return nil
	}

	o.log.Debugf("Download %s archive", o.options.DisplayName)
	archivePath, err := o.download(o.getDownloadFolder(), o.log)
	if err != nil {
		return err
	}

	o.log.Infof("Extract %s...", o.options.DisplayName)
	err = o.extractArchive(archivePath, targetLocation)
	if err != nil {
		return err
	}

	err = copy2.ChownR(path.Join(baseFolder, ".cache"), o.userName)
	if err != nil {
		return errors.Wrap(err, "chown")
	}
	o.log.Infof("Successfully installed %s backend", o.options.DisplayName)
	return nil
}

func getBaseFolder(userName string) (string, error) {
	var err error
	homeFolder := ""
	if userName != "" {
		homeFolder, err = command.GetHome(userName)
	} else {
		homeFolder, err = util.UserHomeDir()
	}
	if err != nil {
		return "", err
	}

	return homeFolder, nil
}

func (o *GenericJetBrainsServer) getDirectory(baseFolder string) string {
	return path.Join(baseFolder, ".cache", "JetBrains", "RemoteDev", "dist", o.options.ID)
}

func (o *GenericJetBrainsServer) extractArchive(fromPath string, toPath string) error {
	file, err := os.Open(fromPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return extract.Extract(file, toPath, extract.StripLevels(1))
}

func (o *GenericJetBrainsServer) download(targetFolder string, log log.Logger) (string, error) {
	err := os.MkdirAll(targetFolder, os.ModePerm)
	if err != nil {
		return "", err
	}

	downloadURL := o.options.DownloadAmd64
	if runtime.GOARCH == "arm64" {
		downloadURL = o.options.DownloadArm64
	}

	targetPath := path.Join(filepath.ToSlash(targetFolder), o.options.ID+".tar.gz")

	// initiate download
	log.Infof("Download %s from %s", o.options.DisplayName, downloadURL)
	defer log.Debugf("Successfully downloaded %s", o.options.DisplayName)
	resp, err := devpodhttp.GetHTTPClient().Get(downloadURL)
	if err != nil {
		return "", errors.Wrap(err, "download binary")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", errors.Wrapf(err, "download binary returned status code %d", resp.StatusCode)
	}

	stat, err := os.Stat(targetPath)
	if err == nil && stat.Size() == resp.ContentLength {
		return targetPath, nil
	}

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
