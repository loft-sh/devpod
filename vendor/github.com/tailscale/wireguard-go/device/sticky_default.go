//go:build !linux

package device

import (
	"github.com/tailscale/wireguard-go/conn"
	"github.com/tailscale/wireguard-go/rwcancel"
)

func (device *Device) startRouteListener(bind conn.Bind) (*rwcancel.RWCancel, error) {
	return nil, nil
}
