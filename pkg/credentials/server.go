package credentials

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	locald "github.com/loft-sh/devpod/pkg/daemon/local"
	workspaced "github.com/loft-sh/devpod/pkg/daemon/workspace"
	network "github.com/loft-sh/devpod/pkg/daemon/workspace/network"
	devpodlog "github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/ts"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	DefaultPort              = "12049"
	CredentialsServerPortEnv = "DEVPOD_CREDENTIALS_SERVER_PORT"
	CredentialsServerLogFile = "devpod-credentials-server.log"
)

// RunCredentialsServer starts a credentials server inside the DevPod workspace.
func RunCredentialsServer(
	ctx context.Context,
	port int,
	client tunnel.TunnelClient,
	clientHost string,
	logger log.Logger,
) error {
	logPath := filepath.Join("/tmp", CredentialsServerLogFile)
	fileLogger := log.NewFileLogger(logPath, logrus.DebugLevel)
	combinedLogger := devpodlog.NewCombinedLogger(logrus.DebugLevel, logger, fileLogger)

	var handler http.Handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		combinedLogger.Debugf("Incoming client connection at %s", request.URL.Path)
		if request.URL.Path == "/git-credentials" {
			err := handleGitCredentialsRequest(ctx, writer, request, client, clientHost, combinedLogger)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
		} else if request.URL.Path == "/docker-credentials" {
			err := handleDockerCredentialsRequest(ctx, writer, request, client, combinedLogger)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
		} else if request.URL.Path == "/git-ssh-signature" {
			err := handleGitSSHSignatureRequest(ctx, writer, request, client, combinedLogger)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
		} else if request.URL.Path == "/loft-platform-credentials" {
			err := handleLoftPlatformCredentialsRequest(ctx, writer, request, client, combinedLogger)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
			}
		} else if request.URL.Path == "/gpg-public-keys" {
			err := handleGPGPublicKeysRequest(ctx, writer, request, client, combinedLogger)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
			}
		}
	})

	addr := net.JoinHostPort("localhost", strconv.Itoa(port))
	srv := &http.Server{Addr: addr, Handler: handler}

	errChan := make(chan error, 1)
	go func() {
		combinedLogger.Debugf("Credentials server started on port %d...", port)

		// always returns error. ErrServerClosed on graceful close
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- err
		} else {
			errChan <- nil
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		_ = srv.Close()
		return nil
	}
}

func GetPort() (int, error) {
	strPort := cmp.Or(os.Getenv(CredentialsServerPortEnv), DefaultPort)
	port, err := strconv.Atoi(strPort)
	if err != nil {
		return 0, fmt.Errorf("convert port %s: %w", strPort, err)
	}

	return port, nil
}

func handleDockerCredentialsRequest(ctx context.Context, writer http.ResponseWriter, request *http.Request, client tunnel.TunnelClient, log log.Logger) error {
	out, err := io.ReadAll(request.Body)
	if err != nil {
		return errors.Wrap(err, "read request body")
	}

	log.Debugf("Received docker credentials post data: %s", string(out))
	response, err := client.DockerCredentials(ctx, &tunnel.Message{Message: string(out)})
	if err != nil {
		return errors.Wrap(err, "get docker credentials response")
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(response.Message))
	log.Debugf("Successfully wrote back %d bytes", len(response.Message))
	return nil
}

func handleGitCredentialsRequest(ctx context.Context, writer http.ResponseWriter, request *http.Request, client tunnel.TunnelClient, clientHost string, log log.Logger) error {
	if clientHost != "" {
		return handleGitCredentialsOverTSNet(ctx, writer, request, clientHost, log)
	}
	out, err := io.ReadAll(request.Body)
	if err != nil {
		return errors.Wrap(err, "read request body")
	}

	log.Debugf("Received git credentials post data: %s", string(out))
	response, err := client.GitCredentials(ctx, &tunnel.Message{Message: string(out)})
	if err != nil {
		log.Debugf("Error receiving git credentials: %v", err)
		return errors.Wrap(err, "get git credentials response")
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(response.Message))
	log.Debugf("Successfully wrote back %d bytes", len(response.Message))
	return nil
}

func handleGitCredentialsOverTSNet(ctx context.Context, writer http.ResponseWriter, request *http.Request, clientHost string, log log.Logger) error {
	bodyBytes, err := io.ReadAll(request.Body)
	if err != nil {
		return errors.Wrap(err, "read request body")
	}
	defer request.Body.Close()

	log.Infof("Received git credentials post data: %s", string(bodyBytes))
	// Set up HTTP transport that uses our network socket.
	socketPath := filepath.Join(workspaced.RootDir, network.TSNetProxySocket)
	transport := &http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second, // TODO: extract this to config
	}

	credServerAddress := ts.EnsureURL(clientHost, locald.LocalCredentialsServerPort)
	targetURL := fmt.Sprintf("http://%s%s", credServerAddress, request.URL.RequestURI())

	// Recreate the request to new targetURL.
	newReq, err := http.NewRequest(request.Method, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		log.Errorf("Failed to create new request: %v", err)
		return errors.Wrap(err, "create request")
	}
	newReq.Header = request.Header.Clone()

	log.Infof("Forwarding request to %s via socket %s", targetURL, socketPath)

	// Execute the request.
	resp, err := client.Do(newReq)
	if err != nil {
		log.Fatalf("HTTP request error: %v", err)
		return errors.Wrap(err, "HTTP request error")
	}
	defer resp.Body.Close()

	// Read the response from the forwarded request.
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
		return errors.Wrap(err, "read response")
	}
	log.Infof("Response: %s", string(respBody))

	// Write the response back to the original response.
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	if _, err := writer.Write(respBody); err != nil {
		log.Errorf("Error writing response to client: %v", err)
		return errors.Wrap(err, "write response")
	}

	return nil
}

func handleGitSSHSignatureRequest(ctx context.Context, writer http.ResponseWriter, request *http.Request, client tunnel.TunnelClient, log log.Logger) error {
	out, err := io.ReadAll(request.Body)
	if err != nil {
		return errors.Wrap(err, "read request body")
	}

	log.Debugf("Received git ssh signature post data: %s", string(out))
	response, err := client.GitSSHSignature(ctx, &tunnel.Message{Message: string(out)})
	if err != nil {
		log.Errorf("Error receiving git ssh signature: %w", err)
		return errors.Wrap(err, "get git ssh signature")
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(response.Message))
	log.Debugf("Successfully wrote back %d bytes", len(response.Message))
	return nil
}

func handleLoftPlatformCredentialsRequest(ctx context.Context, writer http.ResponseWriter, request *http.Request, client tunnel.TunnelClient, log log.Logger) error {
	out, err := io.ReadAll(request.Body)
	if err != nil {
		return errors.Wrap(err, "read request body")
	}

	log.Debugf("Received loft platform credentials post data: %s", string(out))
	response, err := client.LoftConfig(ctx, &tunnel.Message{Message: string(out)})
	if err != nil {
		log.Errorf("Error receiving platform credentials: %w", err)
		return errors.Wrap(err, "get platform credentials")
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(response.Message))
	log.Debugf("Successfully wrote back %d bytes", len(response.Message))
	return nil
}

func handleGPGPublicKeysRequest(ctx context.Context, writer http.ResponseWriter, request *http.Request, client tunnel.TunnelClient, log log.Logger) error {
	response, err := client.GPGPublicKeys(ctx, &tunnel.Message{})
	if err != nil {
		log.Errorf("Error receiving gpg public keys: %w", err)
		return errors.Wrap(err, "get gpg public keys")
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(response.Message))
	log.Debugf("Successfully wrote back %d bytes", len(response.Message))
	return nil
}
