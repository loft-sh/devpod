package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/devpod/pkg/platform"
)

const devPodClientPrefix = 0x01

type LocalClient struct {
	httpClient *http.Client
	provider   string
}

func NewLocalClient(provider string) *LocalClient {
	socketAddr := GetSocketAddr(provider)
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := Dial(socketAddr)
		if err != nil {
			return nil, err
		}
		_, err = conn.Write([]byte{devPodClientPrefix})
		if err != nil {
			return nil, err
		}
		return conn, err
	}
	tr.TLSHandshakeTimeout = 2 * time.Second
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

func (c *LocalClient) GetWorkspace(ctx context.Context, uid string) (*managementv1.DevPodWorkspaceInstance, error) {
	b, err := c.doRequest(ctx, http.MethodGet, routeGetWorkspace+fmt.Sprintf("?uid=%s", uid), nil)
	if err != nil {
		return nil, err
	}

	if len(b) == 0 {
		return nil, nil
	}
	instance := &managementv1.DevPodWorkspaceInstance{}
	err = json.Unmarshal(b, instance)
	if err != nil {
		return nil, err
	}

	return instance, nil
}

func (c *LocalClient) ListWorkspaces(ctx context.Context, ownerFilter platform.OwnerFilter) ([]managementv1.DevPodWorkspaceInstance, error) {
	b, err := c.doRequest(ctx, http.MethodGet, routeListWorkspaces+"?owner="+ownerFilter.String(), nil)
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

func (c *LocalClient) CreateWorkspace(ctx context.Context, workspace *managementv1.DevPodWorkspaceInstance) (*managementv1.DevPodWorkspaceInstance, error) {
	body, err := json.Marshal(workspace)
	if err != nil {
		return nil, err
	}
	b, err := c.doRequest(ctx, http.MethodPost, routeCreateWorkspace, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	newInstance := &managementv1.DevPodWorkspaceInstance{}
	err = json.Unmarshal(b, newInstance)
	if err != nil {
		return nil, err
	}

	return newInstance, nil
}

func (c *LocalClient) UpdateWorkspace(ctx context.Context, workspace *managementv1.DevPodWorkspaceInstance) (*managementv1.DevPodWorkspaceInstance, error) {
	body, err := json.Marshal(workspace)
	if err != nil {
		return nil, err
	}
	b, err := c.doRequest(ctx, http.MethodPost, routeUpdateWorkspace, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	newInstance := &managementv1.DevPodWorkspaceInstance{}
	err = json.Unmarshal(b, newInstance)
	if err != nil {
		return nil, err
	}

	return newInstance, nil
}

func (c *LocalClient) Shutdown(ctx context.Context) error {
	_, err := c.doRequest(ctx, http.MethodGet, routeShutdown, nil)
	if err != nil {
		return err
	}

	return nil
}

func (c *LocalClient) doRequest(ctx context.Context, method string, path string, body io.Reader) ([]byte, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, method, fmt.Sprintf("http://localclient.devpod%s", path), body)
	if err != nil {
		return nil, err
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		if isConnectToDaemonError(err) {
			return nil, &DaemonNotAvailableError{Err: err, Provider: c.provider}
		}

		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%s: %s", res.Status, string(b))
	}

	return io.ReadAll(res.Body)
}
