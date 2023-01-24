package httpclient

import (
	"fmt"
	"net/http"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/hc-install/internal/version"
)

// NewHTTPClient provides a pre-configured http.Client
// e.g. with relevant User-Agent header
func NewHTTPClient() *http.Client {
	client := cleanhttp.DefaultClient()

	userAgent := fmt.Sprintf("hc-install/%s", version.ModuleVersion())

	cli := cleanhttp.DefaultPooledClient()
	cli.Transport = &userAgentRoundTripper{
		userAgent: userAgent,
		inner:     cli.Transport,
	}

	return client
}

type userAgentRoundTripper struct {
	inner     http.RoundTripper
	userAgent string
}

func (rt *userAgentRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if _, ok := req.Header["User-Agent"]; !ok {
		req.Header.Set("User-Agent", rt.userAgent)
	}
	return rt.inner.RoundTrip(req)
}
