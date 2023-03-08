package download

import (
	"crypto/tls"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"net/http"
)

func File(rawURL string) (io.ReadCloser, error) {
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "download file")
	} else if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("received status code %d when trying to download %s", resp.StatusCode, rawURL)
	}

	return resp.Body, nil
}
