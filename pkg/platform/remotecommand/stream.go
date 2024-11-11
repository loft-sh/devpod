package remotecommand

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/gorilla/websocket"
	"k8s.io/klog/v2"
)

func NewStream(ws *WebsocketConn, dataType, closeType MessageType) *Stream {
	return &Stream{
		ws:        ws,
		dataType:  dataType,
		closeType: closeType,
	}
}

type Stream struct {
	ws *WebsocketConn

	dataType  MessageType
	closeType MessageType
}

func (s *Stream) Write(ctx context.Context, writer io.WriteCloser) error {
	if writer == nil {
		return nil
	}

	for {
		_, raw, err := s.ws.ReadMessage()
		if err != nil {
			break
		}

		message, err := ParseMessage(bytes.NewReader(raw))
		if err != nil {
			klog.FromContext(ctx).Error(err, "Unexpected message")
			continue
		}

		if message.messageType == s.dataType {
			if _, err := io.Copy(writer, message.data); err != nil {
				break
			}
		} else if message.messageType == s.closeType {
			return writer.Close()
		}
	}

	return nil
}

func (s *Stream) Read(reader io.Reader) error {
	if reader == nil {
		return s.ws.WriteMessage(websocket.BinaryMessage, newCloseMessage(s.closeType).Bytes())
	}

	buf := make([]byte, MaxMessageSize-1)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			err = s.ws.WriteMessage(websocket.BinaryMessage, newDataMessage(s.dataType, buf[:n]).Bytes())
			if err != nil {
				//nolint:all
				if err == websocket.ErrCloseSent {
					return nil
				}

				return err
			}
		}

		//nolint:all
		if err == io.EOF {
			_ = s.ws.WriteMessage(websocket.BinaryMessage, newCloseMessage(s.closeType).Bytes())
			return nil
		} else if err != nil {
			return fmt.Errorf("read reader: %w", err)
		}
	}
}
