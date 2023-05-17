package machine

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/loft-sh/devpod/cmd/flags"
	devagent "github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/token"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	agent "golang.org/x/crypto/ssh/agent"
	"golang.org/x/term"
)

// SSHCmd holds the configuration
type SSHCmd struct {
	*flags.GlobalFlags

	Command         string
	AgentForwarding bool
}

// NewSSHCmd creates a new destroy command
func NewSSHCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &SSHCmd{
		GlobalFlags: flags,
	}
	sshCmd := &cobra.Command{
		Use:   "ssh",
		Short: "SSH into the machine",
		RunE: func(c *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	sshCmd.Flags().StringVar(&cmd.Command, "command", "", "The command to execute on the remote machine")
	sshCmd.Flags().BoolVar(&cmd.AgentForwarding, "agent-forwarding", false, "If true, will forward the local ssh keys")
	return sshCmd
}

// Run runs the command logic
func (cmd *SSHCmd) Run(ctx context.Context, args []string) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	machineClient, err := workspace.GetMachine(devPodConfig, args, log.Default)
	if err != nil {
		return err
	}

	// get token
	tok, err := token.GetDevPodToken()
	if err != nil {
		return err
	}

	// get private key
	privateKey, err := devssh.GetDevPodPrivateKeyRaw()
	if err != nil {
		return err
	}

	writer := log.Default.ErrorStreamOnly().Writer(logrus.InfoLevel, false)
	defer writer.Close()

	// start the ssh session
	return StartSSHSession(ctx, privateKey, "", cmd.Command, cmd.AgentForwarding, func(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
		command := fmt.Sprintf("'%s' helper ssh-server --token '%s' --stdio", machineClient.AgentPath(), tok)
		if cmd.Debug {
			command += " --debug"
		}
		return devagent.InjectAgentAndExecute(ctx, func(ctx context.Context, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			return machineClient.Command(ctx, client.CommandOptions{
				Command: command,
				Stdin:   stdin,
				Stdout:  stdout,
				Stderr:  stderr,
			})
		}, machineClient.AgentLocal(), machineClient.AgentPath(), machineClient.AgentURL(), true, command, stdin, stdout, stderr, log.Default.ErrorStreamOnly())
	}, writer)
}

type ExecFunc func(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error

func StartSSHSession(ctx context.Context, privateKey []byte, user, command string, agentForwarding bool, exec ExecFunc, stderr io.Writer) error {
	// create readers
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	defer stdoutWriter.Close()
	defer stdinWriter.Close()

	// start ssh machine
	errChan := make(chan error, 1)
	go func() {
		errChan <- exec(ctx, stdinReader, stdoutWriter, stderr)
	}()

	// start ssh client as root / default user
	sshClient, err := devssh.StdioClientFromKeyBytesWithUser(privateKey, stdoutReader, stdinWriter, user, false)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	// create a new session
	session, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	var (
		stdout io.Writer = os.Stdout
		stdin  io.Reader = os.Stdin
	)

	// request agent forwarding
	authSock := os.Getenv("SSH_AUTH_SOCK")
	if agentForwarding && authSock != "" {
		err = agent.ForwardToRemote(sshClient, authSock)
		if err != nil {
			return errors.Errorf("forward agent: %v", err)
		}

		err = agent.RequestAgentForwarding(session)
		if err != nil {
			return errors.Errorf("request agent forwarding: %v", err)
		}
	}

	stdoutFile, validOut := stdout.(*os.File)
	stdinFile, validIn := stdin.(*os.File)
	if validOut && validIn && isatty.IsTerminal(stdoutFile.Fd()) {
		state, err := term.MakeRaw(int(stdinFile.Fd()))
		if err != nil {
			return err
		}
		defer func() {
			_ = term.Restore(int(stdinFile.Fd()), state)
		}()

		windowChange := devssh.WatchWindowSize(ctx)
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-windowChange:
				}
				width, height, err := term.GetSize(int(stdoutFile.Fd()))
				if err != nil {
					continue
				}
				_ = session.WindowChange(height, width)
			}
		}()

		err = session.RequestPty("xterm-256color", 128, 128, ssh.TerminalModes{})
		if err != nil {
			return err
		}
	}

	session.Stdin = stdin
	session.Stdout = stdout
	session.Stderr = stderr
	if command == "" {
		err = session.Shell()
	} else {
		err = session.Start(command)
	}
	if err != nil {
		return err
	}

	// set correct window size
	if validOut && validIn && isatty.IsTerminal(stdoutFile.Fd()) {
		width, height, err := term.GetSize(int(stdoutFile.Fd()))
		if err == nil {
			_ = session.WindowChange(height, width)
		}
	}

	// wait until done
	err = session.Wait()
	if err != nil {
		return err
	}

	return nil
}
