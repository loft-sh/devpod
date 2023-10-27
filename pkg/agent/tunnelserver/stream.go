package tunnelserver

import (
	"errors"
	"io"
	"time"

	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	"github.com/loft-sh/log"
	"github.com/sabhiram/go-gitignore"

	"os"
	"path/filepath"
	"strings"
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

func newStreamWriter(stream tunnel.Tunnel_StreamWorkspaceServer, log log.Logger, gitIgnoreEnabled bool) *streamWriter {
	// Parse the .gitignore file and create an IgnoreParser object
	gitIgnore, err := gitignore.CompileIgnoreFile(".gitignore")
	if err != nil {
		log.Warnf("Failed to parse .gitignore file: %v", err)
		gitIgnore = gitignore.NewGitIgnore() // Use an empty IgnoreParser object if parsing fails
	}

	return &streamWriter{
		stream:           stream,
		bytesWritten:     0,
		lastMessage:      time.Now(),
		log:              log,
		gitIgnore:        gitIgnore, // Set the gitIgnore field to the IgnoreParser object
		gitIgnoreEnabled: gitIgnoreEnabled,
	}
}

type streamWriter struct {
	stream tunnel.Tunnel_StreamWorkspaceServer

	lastMessage  time.Time
	bytesWritten int64
	log          log.Logger
	gitIgnore    gitignore.IgnoreParser
}

func (s *streamWriter) Write(p []byte) (int, error) {
	var filesToUpload []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(".", path)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(relPath, "./") {
			relPath = "./" + relPath
		}
		if !s.gitIgnore.Match(relPath, info.IsDir()) || !s.gitIgnoreEnabled { // Check if gitIgnoreEnabled is true
			filesToUpload = append(filesToUpload, path)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	for _, path := range filesToUpload {
		file, err := os.Open(path)
		if err != nil {
			return 0, err
		}
		defer file.Close()

		buf := make([]byte, 1024)
		for {
			n, err := file.Read(buf)
			if err != nil && err != io.EOF {
				return 0, err
			}
			if n == 0 {
				break
			}

			err = s.stream.Send(&tunnel.Chunk{Content: buf[:n]})
			if err != nil {
				return 0, err
			}

			s.bytesWritten += int64(n)
			if time.Since(s.lastMessage) > time.Second*2 {
				s.log.Infof("Uploaded %.2f MB", float64(s.bytesWritten)/1024/1024)
				s.lastMessage = time.Now()
			}
		}
	}

	return len(p), nil
}
