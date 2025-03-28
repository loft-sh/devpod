package credentials

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
)

const DefaultPort = "12049"
const CredentialsServerPortEnv = "DEVPOD_CREDENTIALS_SERVER_PORT"

func RunCredentialsServer(
	ctx context.Context,
	port int,
	client tunnel.TunnelClient,
	log log.Logger,
) error {
	var handler http.Handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		log.Debugf("Incoming client connection at %s", request.URL.Path)
		if request.URL.Path == "/git-credentials" {
			err := handleGitCredentialsRequest(ctx, writer, request, client, log)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
		} else if request.URL.Path == "/docker-credentials" {
			err := handleDockerCredentialsRequest(ctx, writer, request, client, log)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
		} else if request.URL.Path == "/git-ssh-signature" {
			err := handleGitSSHSignatureRequest(ctx, writer, request, client, log)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
		} else if request.URL.Path == "/loft-platform-credentials" {
			err := handleLoftPlatformCredentialsRequest(ctx, writer, request, client, log)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
			}
		} else if request.URL.Path == "/gpg-public-keys" {
			err := handleGPGPublicKeysRequest(ctx, writer, request, client, log)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
			}
		}
	})

	addr := net.JoinHostPort("localhost", strconv.Itoa(port))
	srv := &http.Server{Addr: addr, Handler: handler}

	errChan := make(chan error, 1)
	go func() {
		log.Debugf("Credentials server started on port %d...", port)

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

func handleGitCredentialsRequest(ctx context.Context, writer http.ResponseWriter, request *http.Request, client tunnel.TunnelClient, log log.Logger) error {
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
