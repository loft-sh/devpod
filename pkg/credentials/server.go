package credentials

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
)

const DefaultPort = "12049"
const DefaultRunnerPort = "12050"
const CredentialsServerPortEnv = "DEVPOD_CREDENTIALS_SERVER_PORT"
const CredentialsServerRunnerPortEnv = "DEVPOD_CREDENTIALS_SERVER_RUNNER_PORT"

func RunCredentialsServer(
	ctx context.Context,
	port int,
	client tunnel.TunnelClient,
	runnerAddr string,
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
		}
	})

	if runnerAddr != "" {
		handler = runnerProxy(handler, runnerAddr, log)
	}

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

func GetRunnerPort() (int, error) {
	strPort := cmp.Or(os.Getenv(CredentialsServerRunnerPortEnv), DefaultRunnerPort)
	port, err := strconv.Atoi(strPort)
	if err != nil {
		return 0, fmt.Errorf("convert port %s: %w", strPort, err)
	}

	return port, nil
}

func runnerProxy(handler http.Handler, proxyAddr string, log log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		proxyReq, err := prepareRequest(req, proxyAddr)
		if err != nil {
			log.Errorf("prepare proxy request", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// execute request against runner
		res, err := devpodhttp.GetHTTPClient().Do(&proxyReq)
		if err != nil {
			log.Errorf("request from proxy: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer res.Body.Close()
		out, err := io.ReadAll(res.Body)
		if err != nil {
			log.Errorf("read response body: %v", err)
			return
		}
		if res.StatusCode != http.StatusOK {
			log.Errorf("proxy request (%d): %d bytes", res.StatusCode, len(out))
			return
		}

		// Send response from runner if it's not empty
		if len(out) != 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(out)
			log.Debugf("Successfully wrote back %d bytes", len(out))
			return
		}

		// Otherwise forward to origin credentials server
		handler.ServeHTTP(w, req)
	})
}

func prepareRequest(req *http.Request, proxyAddr string) (http.Request, error) {
	proxyReq := *req
	var b bytes.Buffer
	_, err := b.ReadFrom(req.Body)
	if err != nil {
		return proxyReq, fmt.Errorf("read body: %w", err)
	}
	req.Body = io.NopCloser(&b)
	proxyReq.Body = io.NopCloser(bytes.NewReader(b.Bytes()))

	// rewrite target
	p, err := url.JoinPath(fmt.Sprintf("http://%s", proxyAddr), req.URL.Path)
	if err != nil {
		return proxyReq, fmt.Errorf("join url path \"http://%s\", \"%s\": %w", proxyAddr, req.URL.Path, err)
	}

	proxyURL, err := url.Parse(p)
	if err != nil {
		return proxyReq, fmt.Errorf("parse proxy url %s: %w", p, err)
	}

	proxyReq.URL = proxyURL
	proxyReq.RequestURI = ""

	return proxyReq, nil
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
		log.Errorf("Error receiving git ssh signature: %w", err)
		return errors.Wrap(err, "get git ssh signature")
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(response.Message))
	log.Debugf("Successfully wrote back %d bytes", len(response.Message))
	return nil
}
