package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/pkg/extract"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/scanner"
	"github.com/loft-sh/devpod/pkg/survey"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"net"
	"os"
	"strings"

	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/stdio"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func NewTunnelClient(reader io.Reader, writer io.WriteCloser, exitOnClose bool) (tunnel.TunnelClient, error) {
	pipe := stdio.NewStdioStream(reader, writer, exitOnClose)

	// Set up a connection to the server.
	conn, err := grpc.Dial("", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		return pipe, nil
	}))
	if err != nil {
		return nil, err
	}

	return tunnel.NewTunnelClient(conn), nil
}

func StartTunnelServer(reader io.Reader, writer io.WriteCloser, exitOnClose bool, workspace *provider2.Workspace, log log.Logger) error {
	pipe := stdio.NewStdioStream(reader, writer, exitOnClose)
	lis := stdio.NewStdioListener()
	done := make(chan error)

	go func() {
		s := grpc.NewServer()
		tunnelServ := &tunnelServer{
			workspace: workspace,
			log:       log,
		}
		tunnel.RegisterTunnelServer(s, tunnelServ)
		reflection.Register(s)
		done <- s.Serve(lis)
	}()

	lis.Ready(pipe)
	return <-done
}

type tunnelServer struct {
	tunnel.UnimplementedTunnelServer

	workspace *provider2.Workspace
	log       log.Logger
}

func (t *tunnelServer) Workspace(context.Context, *tunnel.Empty) (*tunnel.WorkspaceInfo, error) {
	out, err := json.Marshal(t.workspace)
	if err != nil {
		return nil, errors.Wrap(err, "marshal workspace")
	}

	return &tunnel.WorkspaceInfo{Workspace: string(out)}, nil
}

func (t *tunnelServer) Ping(context.Context, *tunnel.Empty) (*tunnel.Empty, error) {
	return &tunnel.Empty{}, nil
}

func (t *tunnelServer) Log(ctx context.Context, message *tunnel.LogMessage) (*tunnel.Empty, error) {
	if message.LogLevel == tunnel.LogLevel_DEBUG {
		t.log.Debug(strings.TrimSpace(message.Message))
	} else if message.LogLevel == tunnel.LogLevel_INFO {
		t.log.Info(strings.TrimSpace(message.Message))
	} else if message.LogLevel == tunnel.LogLevel_WARNING {
		t.log.Warn(strings.TrimSpace(message.Message))
	} else if message.LogLevel == tunnel.LogLevel_ERROR {
		t.log.Error(strings.TrimSpace(message.Message))
	} else if message.LogLevel == tunnel.LogLevel_DONE {
		t.log.Done(strings.TrimSpace(message.Message))
	}

	return &tunnel.Empty{}, nil
}

func (t *tunnelServer) ReadWorkspace(response *tunnel.Empty, stream tunnel.Tunnel_ReadWorkspaceServer) error {
	return extract.WriteTar(NewStreamWriter(stream), t.workspace.Source.LocalFolder)
}

func NewStreamReader(stream tunnel.Tunnel_ReadWorkspaceClient) io.Reader {
	reader, writer := io.Pipe()
	go func() {
		defer reader.Close()
		defer writer.Close()

		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				_ = writer.CloseWithError(err)
			} else if len(resp.Content) > 0 {
				_, err = writer.Write(resp.Content)
				if err != nil {
					_ = writer.CloseWithError(err)
				}
			}
		}
	}()

	return reader
}

func NewStreamWriter(stream tunnel.Tunnel_ReadWorkspaceServer) io.Writer {
	return &streamWriter{stream: stream}
}

type streamWriter struct {
	stream tunnel.Tunnel_ReadWorkspaceServer
}

func (s *streamWriter) Write(p []byte) (int, error) {
	err := s.stream.Send(&tunnel.Chunk{Content: p})
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

func NewTunnelLogger(ctx context.Context, client tunnel.TunnelClient, debug bool) log.Logger {
	level := logrus.InfoLevel
	if debug {
		level = logrus.DebugLevel
	}

	return &tunnelLogger{ctx: ctx, client: client, level: level}
}

type tunnelLogger struct {
	ctx    context.Context
	level  logrus.Level
	client tunnel.TunnelClient
}

func (s *tunnelLogger) Debug(args ...interface{}) {
	if s.level < logrus.DebugLevel {
		return
	}

	_, _ = s.client.Log(s.ctx, &tunnel.LogMessage{
		LogLevel: tunnel.LogLevel_DEBUG,
		Message:  fmt.Sprintln(args...),
	})
}

func (s *tunnelLogger) Debugf(format string, args ...interface{}) {
	if s.level < logrus.DebugLevel {
		return
	}

	_, _ = s.client.Log(s.ctx, &tunnel.LogMessage{
		LogLevel: tunnel.LogLevel_DEBUG,
		Message:  fmt.Sprintf(format, args...) + "\n",
	})
}

func (s *tunnelLogger) Info(args ...interface{}) {
	if s.level < logrus.InfoLevel {
		return
	}

	_, _ = s.client.Log(s.ctx, &tunnel.LogMessage{
		LogLevel: tunnel.LogLevel_INFO,
		Message:  fmt.Sprintln(args...),
	})
}

func (s *tunnelLogger) Infof(format string, args ...interface{}) {
	if s.level < logrus.InfoLevel {
		return
	}

	_, _ = s.client.Log(s.ctx, &tunnel.LogMessage{
		LogLevel: tunnel.LogLevel_INFO,
		Message:  fmt.Sprintf(format, args...) + "\n",
	})
}

func (s *tunnelLogger) Warn(args ...interface{}) {
	if s.level < logrus.WarnLevel {
		return
	}

	_, _ = s.client.Log(s.ctx, &tunnel.LogMessage{
		LogLevel: tunnel.LogLevel_WARNING,
		Message:  fmt.Sprintln(args...),
	})
}

func (s *tunnelLogger) Warnf(format string, args ...interface{}) {
	if s.level < logrus.WarnLevel {
		return
	}

	_, _ = s.client.Log(s.ctx, &tunnel.LogMessage{
		LogLevel: tunnel.LogLevel_WARNING,
		Message:  fmt.Sprintf(format, args...) + "\n",
	})
}

func (s *tunnelLogger) Error(args ...interface{}) {
	if s.level < logrus.ErrorLevel {
		return
	}

	_, _ = s.client.Log(s.ctx, &tunnel.LogMessage{
		LogLevel: tunnel.LogLevel_ERROR,
		Message:  fmt.Sprintln(args...),
	})
}

func (s *tunnelLogger) Errorf(format string, args ...interface{}) {
	if s.level < logrus.ErrorLevel {
		return
	}

	_, _ = s.client.Log(s.ctx, &tunnel.LogMessage{
		LogLevel: tunnel.LogLevel_ERROR,
		Message:  fmt.Sprintf(format, args...) + "\n",
	})
}

func (s *tunnelLogger) Fatal(args ...interface{}) {
	if s.level < logrus.FatalLevel {
		return
	}

	_, _ = s.client.Log(s.ctx, &tunnel.LogMessage{
		LogLevel: tunnel.LogLevel_ERROR,
		Message:  fmt.Sprintln(args...),
	})

	os.Exit(1)
}

func (s *tunnelLogger) Fatalf(format string, args ...interface{}) {
	if s.level < logrus.FatalLevel {
		return
	}

	_, _ = s.client.Log(s.ctx, &tunnel.LogMessage{
		LogLevel: tunnel.LogLevel_ERROR,
		Message:  fmt.Sprintf(format, args...) + "\n",
	})

	os.Exit(1)
}

func (s *tunnelLogger) Done(args ...interface{}) {
	if s.level < logrus.InfoLevel {
		return
	}

	_, _ = s.client.Log(s.ctx, &tunnel.LogMessage{
		LogLevel: tunnel.LogLevel_DONE,
		Message:  fmt.Sprintln(args...),
	})
}

func (s *tunnelLogger) Donef(format string, args ...interface{}) {
	if s.level < logrus.InfoLevel {
		return
	}

	_, _ = s.client.Log(s.ctx, &tunnel.LogMessage{
		LogLevel: tunnel.LogLevel_DONE,
		Message:  fmt.Sprintf(format, args...) + "\n",
	})
}

func (s *tunnelLogger) Print(level logrus.Level, args ...interface{}) {
	switch level {
	case logrus.InfoLevel:
		s.Info(args...)
	case logrus.DebugLevel:
		s.Debug(args...)
	case logrus.WarnLevel:
		s.Warn(args...)
	case logrus.ErrorLevel:
		s.Error(args...)
	case logrus.FatalLevel:
		s.Fatal(args...)
	}
}

func (s *tunnelLogger) Printf(level logrus.Level, format string, args ...interface{}) {
	switch level {
	case logrus.InfoLevel:
		s.Infof(format, args...)
	case logrus.DebugLevel:
		s.Debugf(format, args...)
	case logrus.WarnLevel:
		s.Warnf(format, args...)
	case logrus.ErrorLevel:
		s.Errorf(format, args...)
	case logrus.FatalLevel:
		s.Fatalf(format, args...)
	}
}

func (s *tunnelLogger) SetLevel(level logrus.Level) {
	s.level = level
}

func (s *tunnelLogger) GetLevel() logrus.Level {
	return s.level
}

func (s *tunnelLogger) Writer(level logrus.Level, raw bool) io.WriteCloser {
	if s.level < level {
		return &log.NopCloser{Writer: io.Discard}
	}

	reader, writer := io.Pipe()
	go func() {
		sa := scanner.NewScanner(reader)
		for sa.Scan() {
			if raw {
				s.WriteString(level, sa.Text()+"\n")
			} else {
				s.Print(level, sa.Text())
			}
		}
	}()

	return writer
}

func (s *tunnelLogger) WriteString(level logrus.Level, message string) {
	if s.level < level {
		return
	}

	// TODO: support this correctly
	s.Print(level, message)
}

func (s *tunnelLogger) Question(params *survey.QuestionOptions) (string, error) {
	return "", fmt.Errorf("not supported")
}

func (s *tunnelLogger) ErrorStreamOnly() log.Logger {
	return s
}
