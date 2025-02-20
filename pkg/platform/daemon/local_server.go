package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"tailscale.com/net/memnet"

	"tailscale.com/client/tailscale"
	"tailscale.com/ipn"
)

type localServer struct {
	httpServer *http.Server
	lc         *tailscale.LocalClient
	listener   *memnet.Listener
}

type Status struct {
	State DaemonState `json:"state,omitempty"`
}

type DaemonState string

var (
	DaemonStateRunning DaemonState = "running"
	DaemonStateStopped DaemonState = "stopped"
	DaemonStatePending DaemonState = "pending"
)

var (
	routeHealth  = "/health"
	routeMetrics = "/metrics"
	routeStatus  = "/status"
)

func getLocalServer(lc *tailscale.LocalClient) (*localServer, error) {
	l := &localServer{lc: lc}
	m := http.NewServeMux()
	m.HandleFunc("GET "+routeHealth, l.health)
	m.HandleFunc("GET "+routeMetrics, l.metrics)
	m.HandleFunc("GET "+routeStatus, l.status)

	l.httpServer = &http.Server{Handler: m}
	l.listener = memnet.Listen("localclient.devpod:80")

	return l, nil
}

func (l *localServer) ListenAndServe() error {
	return l.httpServer.Serve(l.listener)
}

func (l *localServer) Addr() string {
	return l.listener.Addr().String()
}

func (l *localServer) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	return l.listener.Dial(ctx, network, addr)
}

func (l *localServer) metrics(w http.ResponseWriter, r *http.Request) {
	// TODO: Get from local client
}

func (l *localServer) health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (l *localServer) status(w http.ResponseWriter, r *http.Request) {
	st, err := l.lc.Status(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	daemonState := DaemonStateStopped
	switch st.BackendState {
	case ipn.Starting.String():
		daemonState = DaemonStatePending
	case ipn.Running.String():
		daemonState = DaemonStateRunning
	default:
		// we consider all other states as `stopped`
	}

	status := Status{State: daemonState}
	tryJSON(w, status)
}

func (l *localServer) handleError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}
}

func tryJSON(w http.ResponseWriter, obj interface{}) {
	out, err := json.Marshal(obj)
	if err != nil {
		http.Error(w, fmt.Errorf("marshal: %w", err).Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}
