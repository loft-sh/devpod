package tunnelserver

import (
	"errors"
	"io"
	"time"

	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	"github.com/loft-sh/log"
)

func NewStreamReader(stream tunnel.Tunnel_StreamWorkspaceClient, log log.Logger) io.Reader {
	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()

		for {
			resp, err := stream.Recv()
			if resp != nil && len(resp.Content) > 0 {
				_, err = writer.Write(resp.Content)
				if err != nil {
					log.Debugf("Error writing to pipe: %v", err)
					return
				}
			}
			if errors.Is(err, io.EOF) {
				return
			} else if err != nil {
				log.Debugf("Error receiving from stream: %v", err)
				return
			}
		}
	}()

	return reader
}

func NewStreamWriter(stream tunnel.Tunnel_StreamWorkspaceServer, log log.Logger) io.Writer {
	return &streamWriter{stream: stream, log: log, lastMessage: time.Now()}
}

type streamWriter struct {
	stream tunnel.Tunnel_StreamWorkspaceServer

	lastMessage  time.Time
	bytesWritten int64
	log          log.Logger
}

func (s *streamWriter) Write(p []byte) (int, error) {
	err := s.stream.Send(&tunnel.Chunk{Content: p})
	if err != nil {
		return 0, err
	}

	s.bytesWritten += int64(len(p))
	if time.Since(s.lastMessage) > time.Second*2 {
		s.log.Infof("Uploaded %.2f MB", float64(s.bytesWritten)/1024/1024)
		s.lastMessage = time.Now()
	}

	return len(p), nil
}
