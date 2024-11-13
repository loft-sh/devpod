package util

import (
	"io"
	"net/http"
	"time"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/log"
	"github.com/mitchellh/go-homedir"
)

func GetBaseFolder(userName string) (string, error) {
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

type ProgressReader struct {
	reader io.Reader

	lastMessage time.Time
	bytesRead   int64
	totalSize   int64
	log         log.Logger
}

func (p *ProgressReader) Read(b []byte) (n int, err error) {
	n, err = p.reader.Read(b)
	p.bytesRead += int64(n)
	if time.Since(p.lastMessage) > time.Second*1 {
		p.log.Infof("Downloaded %.2f / %.2f MB", float64(p.bytesRead)/1024/1024, float64(p.totalSize/1024/1024))
		p.lastMessage = time.Now()
	}

	return n, err
}

func NewProgressReader(resp *http.Response, log log.Logger) *ProgressReader {
	return &ProgressReader{
		reader:    resp.Body,
		totalSize: resp.ContentLength,
		log:       log,
	}
}
