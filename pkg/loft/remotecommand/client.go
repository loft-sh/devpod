package remotecommand

import (
	"bytes"
	"context"
	"io"

	"github.com/gorilla/websocket"
	"github.com/loft-sh/log"
)

func ExecuteConn(ctx context.Context, rawConn *websocket.Conn, stdin io.Reader, stdout io.Writer, stderr io.Writer, log log.Logger) (int, error) {
	conn := NewWebsocketConn(rawConn)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// close websocket connection
	defer conn.Close()
	defer func() {
		err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Debugf("error write close: %v", err)
			return
		}
	}()

	// ping connection
	go func() {
		Ping(ctx, conn)
	}()

	// pipe stdout into websocket
	go func() {
		err := NewStream(conn, StdinData, StdinClose).Read(stdin)
		if err != nil {
			log.Debugf("error pipe stdin: %v", err)
		}
	}()

	// read messages
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			log.Debugf("error read message: %v", err)
			return 0, err
		}

		message, err := ParseMessage(bytes.NewReader(raw))
		if err != nil {
			log.Debugf("error parse message: %v", err)
			continue
		}

		if message.messageType == StdoutData {
			if _, err := io.Copy(stdout, message.data); err != nil {
				log.Debugf("error read stdout: %v", err)
				return 1, err
			}
		} else if message.messageType == StderrData {
			if _, err := io.Copy(stderr, message.data); err != nil {
				log.Debugf("error read stderr: %v", err)
				return 1, err
			}
		} else if message.messageType == ExitCode {
			log.Debugf("exit code: %d", message.exitCode)
			return int(message.exitCode), nil
		}
	}
}
