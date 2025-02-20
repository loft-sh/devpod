package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
)

const devPodClientPrefix = 0x01

type Client struct {
	httpClient *http.Client
}

func NewClient(daemonFolder, provider string) Client {
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

	return Client{httpClient: httpClient}
}

func (c *Client) Status(ctx context.Context) (Status, error) {
	b, err := c.doRequest(ctx, http.MethodGet, routeStatus, nil)
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

func (c *Client) doRequest(ctx context.Context, method string, path string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, fmt.Sprintf("http://localclient.devpod%s", path), body)
	if err != nil {
		return nil, err
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	return io.ReadAll(res.Body)
}
