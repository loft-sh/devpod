package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
)

const devPodClientPrefix = 0x01

type LocalClient struct {
	httpClient *http.Client
	provider   string
}

func NewLocalClient(daemonFolder, provider string) *LocalClient {
	socketAddr := GetSocketAddr(daemonFolder, provider)
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := dial(socketAddr)
		if err != nil {
			return nil, err
		}
		_, err = conn.Write([]byte{devPodClientPrefix})
		if err != nil {
			return nil, err
		}
		return conn, err
	}
	httpClient := &http.Client{Transport: tr}

	return &LocalClient{httpClient: httpClient, provider: provider}
}

func (c *LocalClient) Status(ctx context.Context, debug bool) (Status, error) {
	path := routeStatus
	if debug {
		path += "?debug"
	}
	b, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return Status{}, err
	}

	status := Status{}
	err = json.Unmarshal(b, &status)
	if err != nil {
		return Status{}, err
	}

	return status, nil
}

func (c *LocalClient) ListWorkspaces(ctx context.Context) ([]managementv1.DevPodWorkspaceInstance, error) {
	b, err := c.doRequest(ctx, http.MethodGet, routeListWorkspaces, nil)
	if err != nil {
		return nil, err
	}

	instances := []managementv1.DevPodWorkspaceInstance{}
	err = json.Unmarshal(b, &instances)
	if err != nil {
		return nil, err
	}

	return instances, nil
}

func (c *LocalClient) Shutdown(ctx context.Context) error {
	_, err := c.doRequest(ctx, http.MethodGet, routeShutdown, nil)
	if err != nil {
		return err
	}

	return nil
}

func (c *LocalClient) doRequest(ctx context.Context, method string, path string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, fmt.Sprintf("http://localclient.devpod%s", path), body)
	if err != nil {
		return nil, err
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		if isConnectToDaemonError(err) {
			return nil, errDaemonNotAvailable{Err: err, Provider: c.provider}
		}

		return nil, err
	}
	defer res.Body.Close()

	return io.ReadAll(res.Body)
}
