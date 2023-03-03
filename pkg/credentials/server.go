package credentials

import (
	"context"
	"encoding/json"
	"github.com/gofrs/flock"
	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	"github.com/loft-sh/devpod/pkg/dockercredentials"
	"github.com/loft-sh/devpod/pkg/gitcredentials"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

func RunCredentialsServer(ctx context.Context, userName string, port int, configureGitHelper, configureDockerHelper bool, client tunnel.TunnelClient, log log.Logger) error {
	if configureGitHelper || configureDockerHelper {
		fileLock := flock.New(filepath.Join(os.TempDir(), "devpod-credentials.lock"))
		locked, err := fileLock.TryLock()
		if err != nil {
			return errors.Wrap(err, "acquire lock")
		} else if !locked {
			return nil
		}
		defer fileLock.Unlock()

		binaryPath, err := os.Executable()
		if err != nil {
			return err
		}
		if configureDockerHelper {
			// configure the creds store
			err = dockercredentials.ConfigureCredentialsContainer(userName, port)
			if err != nil {
				return err
			}
		}
		if configureGitHelper {
			// set user & email
			response, err := client.GitUser(ctx, &tunnel.Empty{})
			if err != nil {
				log.Errorf("Retrieve git user: %v", err)
			} else {
				gitUser := &gitcredentials.GitUser{}
				err = json.Unmarshal([]byte(response.Message), gitUser)
				if err != nil {
					return err
				}

				err = gitcredentials.SetUser(userName, gitUser)
				if err != nil {
					log.Errorf("Set git user: %v", err)
				}
			}

			// configure helper
			err = gitcredentials.ConfigureHelper(binaryPath, userName, port)
			if err != nil {
				return errors.Wrap(err, "configure git helper")
			}

			// cleanup when we are done
			defer gitcredentials.RemoveHelper(userName)
		}
	}

	srv := &http.Server{
		Addr: "localhost:" + strconv.Itoa(port),
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
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
			}
		}),
	}

	errChan := make(chan error, 1)
	go func() {
		log.Infof("Credentials server started on port %d...", port)

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
		return errors.Wrap(err, "get git credentials response")
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(response.Message))
	log.Debugf("Successfully wrote back %d bytes", len(response.Message))
	return nil
}
