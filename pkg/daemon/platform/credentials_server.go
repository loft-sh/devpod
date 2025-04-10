package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/loft-sh/devpod/pkg/gitcredentials"
	"github.com/loft-sh/log"
	"tailscale.com/tsnet"
)

const (
	LocalCredentialsServerPort = 9999
)

// LocalCredentialsServer runs locally on the user's workspace and listens on tsnet
// for credentials requests.
type LocalCredentialsServer struct {
	log      log.Logger
	tsServer *tsnet.Server

	ln  net.Listener
	srv *http.Server
}

// NewLocalCredentialsServer initializes a new LocalCredentialsServer.
func NewLocalCredentialsServer(tsServer *tsnet.Server, log log.Logger) (*LocalCredentialsServer, error) {
	return &LocalCredentialsServer{
		log:      log,
		tsServer: tsServer,
	}, nil
}

func (s *LocalCredentialsServer) Listen(ctx context.Context) error {
	s.log.Info("Starting credentials server")

	// Create a tsnet listener for LocalCredentialsServer.
	ln, err := s.tsServer.Listen("tcp", fmt.Sprintf(":%d", LocalCredentialsServerPort))
	if err != nil {
		s.log.Infof("Failed to listen on tsnet port %d: %v", LocalCredentialsServerPort, err)
		return fmt.Errorf("failed to listen on tsnet port %d: %w", LocalCredentialsServerPort, err)
	}
	s.ln = ln

	// Create HTTP server and register handlers.
	mux := http.NewServeMux()
	mux.HandleFunc("/git-credentials", s.handleGitCredentials)

	// Create the HTTP server.
	s.srv = &http.Server{
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		s.log.Info("Context canceled, shutting down credentials server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.srv.Shutdown(shutdownCtx); err != nil {
			s.log.Errorf("Error shutting down credentials server: %v", err)
		}
	}()

	s.log.Infof("Credentials server listening on tsnet port %d", LocalCredentialsServerPort)
	err = s.srv.Serve(ln)
	if err != nil && err != http.ErrServerClosed {
		s.log.Errorf("Credentials server error: %v", err)
		return err
	}

	return nil
}

func (s *LocalCredentialsServer) handleGitCredentials(w http.ResponseWriter, r *http.Request) {
	s.log.Infof("Handling git credentials request")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.log.Errorf("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var creds gitcredentials.GitCredentials
	if err := json.Unmarshal(body, &creds); err != nil {
		s.log.Errorf("Error unmarshaling git credentials: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	s.log.Debugf("Parsed Git Credentials request: %+v", creds)

	credentials, err := gitcredentials.GetCredentials(&creds)
	if err != nil {
		s.log.Errorf("Error getting git credentials: %v", err)
		http.Error(w, "Failed to get git credentials", http.StatusInternalServerError)
		return
	}

	responseData, err := json.Marshal(credentials)
	if err != nil {
		s.log.Errorf("Error marshaling credentials response: %v", err)
		http.Error(w, "Failed to serialize credentials", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	s.log.Infof("Sending response: %s", string(responseData))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(responseData)
}

// Close shuts down the credentials server.
func (s *LocalCredentialsServer) Close() error {
	s.log.Info("Closing credentials server")
	if s.srv != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.srv.Shutdown(shutdownCtx); err != nil {
			s.log.Errorf("Error during credentials server shutdown: %v", err)
			return err
		}
	}
	return nil
}
