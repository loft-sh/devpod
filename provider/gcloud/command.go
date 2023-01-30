package gcloud

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/provider/types"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/ssh/server"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"net"
	"time"
)

func (g *gcloudProvider) RunCommand(ctx context.Context, workspace *config.Workspace, options types.RunCommandOptions) error {
	// wait until instance is ready
	name := getName(workspace)
	status, err := g.getWorkspaceStatus(ctx, name)
	if err != nil {
		return errors.Wrap(err, "get instance status")
	} else if status == nil {
		return fmt.Errorf("instance %s not found", name)
	}

	// check if instance has an external ip
	if len(status.NetworkInterfaces) == 0 || len(status.NetworkInterfaces[0].AccessConfigs) == 0 || status.NetworkInterfaces[0].AccessConfigs[0].NatIP == "" {
		// via cmd
		return g.doSSHCommand(ctx, name, options)
	}

	// dial external address
	externalAddress := status.NetworkInterfaces[0].AccessConfigs[0].NatIP
	d := net.Dialer{Timeout: time.Second * 5}
	conn, err := d.Dial("tcp", fmt.Sprintf("%s:%d", externalAddress, server.DefaultPort))
	if err != nil {
		return err
	}

	// get token
	key, err := devssh.GetPrivateKeyRaw(workspace.ID)
	if err != nil {
		return errors.Wrap(err, "read private key")
	}

	// parse private key
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return errors.Wrap(err, "parse private key")
	}

	// create ssh client
	client, err := devssh.CreateFromConn(conn, name, &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		return errors.Wrap(err, "dial agent")
	}
	defer client.Close()

	sess, err := client.NewSession()
	if err != nil {
		return errors.Wrap(err, "create session")
	}
	defer sess.Close()

	sess.Stdin = options.Stdin
	sess.Stdout = options.Stdout
	sess.Stderr = options.Stderr
	return sess.Run(options.Command)
}

func (g *gcloudProvider) doSSHCommand(ctx context.Context, name string, options types.RunCommandOptions) error {
	args := []string{
		"compute",
		"ssh",
		name,
		"--project=" + g.Config.Project,
		"--zone=" + g.Config.Zone,
		"--command",
		options.Command,
	}

	return g.runCommand(ctx, args, options.Stdout, options.Stderr, options.Stdin)
}
