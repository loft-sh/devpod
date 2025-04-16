package network

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/loft-sh/devpod/pkg/ts"
	"github.com/loft-sh/log"
	"tailscale.com/tsnet"
)

// HttpProxyHandler handles the logic for proxying HTTP requests.
type HttpProxyHandler struct {
	log      log.Logger
	tsServer *tsnet.Server
}

// NewHttpProxyHandler creates a new HttpProxyHandler.
func NewHttpProxyHandler(tsSrv *tsnet.Server, logger log.Logger) *HttpProxyHandler {
	return &HttpProxyHandler{
		log:      logger,
		tsServer: tsSrv,
	}
}

func (h *HttpProxyHandler) tsDialer(ctx context.Context, addr string) (net.Conn, error) {
	h.log.Debugf("HttpProxyHandler: Dialing target %s via tsnet", addr)
	conn, err := h.tsServer.Dial(ctx, "tcp", addr)
	if err != nil {
		h.log.Errorf("HttpProxyHandler: Failed to dial target %s via tsnet: %v", addr, err)
		return nil, fmt.Errorf("tsnet dial failed for %s: %w", addr, err)
	}
	h.log.Debugf("HttpProxyHandler: Successfully dialed %s via tsnet", addr)
	return conn, nil
}

// ServeHTTP implements the http.Handler interface.
func (h *HttpProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	targetHostHeader := r.Header.Get(HeaderTargetHost)
	proxyPortHeader := r.Header.Get(HeaderProxyPort)

	var targetAddr string

	if targetHostHeader != "" && proxyPortHeader != "" {
		proxyPort, err := strconv.Atoi(proxyPortHeader)
		if err != nil {
			h.log.Errorf("NetworkProxyService: HTTP: Invalid X-Proxy-Port %q: %v", proxyPortHeader, err)
			http.Error(w, "Invalid X-Proxy-Port header", http.StatusBadRequest)
			return
		}
		targetAddr = ts.EnsureURL(targetHostHeader, proxyPort)
		h.log.Debugf("NetworkProxyService: HTTP: Proxying request for %s %s via custom headers to target %s", r.Method, r.URL.Path, targetAddr)

	} else {
		host := r.Host
		if host == "" {
			h.log.Errorf("NetworkProxyService: HTTP: Request missing Host header")
			http.Error(w, "Host header is required", http.StatusBadRequest)
			return
		}

		if !strings.Contains(host, ":") {
			h.log.Debugf("NetworkProxyService: HTTP: Host header %q missing port, assuming port 80", host)
			targetAddr = net.JoinHostPort(host, "80")
		} else {
			targetAddr = host
		}

		h.log.Debugf("NetworkProxyService: HTTP: Proxying request for %s %s to target %s", r.Method, r.URL.Path, targetAddr)
	}

	dialTargetAddr := targetAddr

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			targetURL := url.URL{
				Scheme: "http",
				Host:   dialTargetAddr,
			}
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.URL.Path = r.URL.Path
			req.URL.RawQuery = r.URL.RawQuery
			req.Host = targetURL.Host
			h.log.Debugf("NetworkProxyService: HTTP: Director setting outgoing URL to %s, Host header to %s", req.URL.String(), req.Host)
		},
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return h.tsDialer(ctx, dialTargetAddr)
			},
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			h.log.Errorf("NetworkProxyService: HTTP: Proxy error to target %s: %v", dialTargetAddr, err)
			http.Error(w, fmt.Sprintf("Proxy error: %v", err), http.StatusBadGateway)
		},
		ModifyResponse: func(resp *http.Response) error {
			h.log.Debugf("NetworkProxyService: HTTP: Received response %d from target %s", resp.StatusCode, dialTargetAddr)
			return nil
		},
	}

	proxy.ServeHTTP(w, r)
}
