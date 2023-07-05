package http

import (
	"crypto/tls"
	"net/http"
	"sync"
)

var httpClient *http.Client
var httpClientOnce sync.Once

func GetHTTPClient() *http.Client {
	httpClientOnce.Do(func() {
		customTransport := http.DefaultTransport.(*http.Transport).Clone()
		customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		httpClient = &http.Client{Transport: customTransport}
	})

	return httpClient
}
