package remotecommand

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	PingWaitDuration = 60 * time.Second
)

func NewWebsocketConn(ws *websocket.Conn) *WebsocketConn {
	conn := &WebsocketConn{
		ws: ws,
	}
	conn.setupDeadline()
	return conn
}

type WebsocketConn struct {
	m sync.Mutex

	ws *websocket.Conn

	closeOnce  sync.Once
	closeError error
}

func (w *WebsocketConn) setupDeadline() {
	_ = w.ws.SetReadDeadline(time.Now().Add(PingWaitDuration))
	w.ws.SetPingHandler(func(string) error {
		w.m.Lock()
		err := w.ws.WriteControl(websocket.PongMessage, []byte(""), time.Now().Add(PingWaitDuration))
		w.m.Unlock()
		if err != nil {
			return err
		}
		if err := w.ws.SetReadDeadline(time.Now().Add(PingWaitDuration)); err != nil {
			return err
		}
		return w.ws.SetWriteDeadline(time.Now().Add(PingWaitDuration))
	})
	w.ws.SetPongHandler(func(string) error {
		if err := w.ws.SetReadDeadline(time.Now().Add(PingWaitDuration)); err != nil {
			return err
		}
		return w.ws.SetWriteDeadline(time.Now().Add(PingWaitDuration))
	})
}

func (w *WebsocketConn) ReadMessage() (messageType int, p []byte, err error) {
	return w.ws.ReadMessage()
}

func (w *WebsocketConn) WriteControl(messageType int, data []byte, deadline time.Time) error {
	w.m.Lock()
	defer w.m.Unlock()

	_ = w.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return w.ws.WriteControl(messageType, data, deadline)
}

func (w *WebsocketConn) WriteMessage(messageType int, data []byte) error {
	w.m.Lock()
	defer w.m.Unlock()

	_ = w.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return w.ws.WriteMessage(messageType, data)
}

func (w *WebsocketConn) Close() error {
	w.closeOnce.Do(func() {
		w.closeError = w.ws.Close()
	})

	return w.closeError
}
