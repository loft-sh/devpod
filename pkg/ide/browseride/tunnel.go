package browseride

import (
	"context"
	"io"

	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/ssh/sshcmd"
	"github.com/loft-sh/devpod/pkg/tunnel"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

func startBrowserTunnel(
	ctx context.Context,
	devPodConfig *config.Config,
	client client2.BaseWorkspaceClient,
	user, targetURL string,
	forwardPorts bool,
	extraPorts []string,
	gitUsername, gitToken string,
	logger log.Logger,
) error {
	err := tunnel.NewTunnel(
		ctx,
		func(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
			writer := logger.Writer(logrus.DebugLevel, false)
			defer writer.Close()

			cmd, err := sshcmd.New(ctx, client, logger, []string{
				"--log-output=raw",
				"--stdio",
			})
			if err != nil {
				return err
			}
			cmd.Stdout = stdout
			cmd.Stdin = stdin
			cmd.Stderr = writer
			return cmd.Run()
		},
		func(ctx context.Context, containerClient *ssh.Client) error {
			// print port to console
			streamLogger, ok := logger.(*log.StreamLogger)
			if ok {
				streamLogger.JSON(logrus.InfoLevel, map[string]string{
					"url":  targetURL,
					"done": "true",
				})
			}

			// run in container
			err := tunnel.RunInContainer(
				ctx,
				devPodConfig,
				containerClient,
				user,
				forwardPorts,
				extraPorts,
				gitUsername,
				gitToken,
				logger,
			)
			if err != nil {
				logger.Errorf("error running credentials server: %v", err)
			}

			<-ctx.Done()
			return nil
		},
	)
	if err != nil {
		return err
	}

	return nil
}
