package remotecommand

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
	"k8s.io/klog/v2"
)

const (
	// Maximum message size allowed from peer.
	MaxMessageSize = 2 << 14
)

func Ping(ctx context.Context, ws *WebsocketConn) {
	defer ws.Close()

	for {
		select {
		case <-time.After(time.Second * 10):
			if err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				klog.FromContext(ctx).Error(err, "Error sending ping message")
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
