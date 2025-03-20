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
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	devsshagent "github.com/loft-sh/devpod/pkg/ssh/agent"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
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
		Use:   "ssh [name]",
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

	writer := log.Default.ErrorStreamOnly().Writer(logrus.InfoLevel, false)
	defer writer.Close()

	// Get the timeout from the context options
	timeout := config.ParseTimeOption(devPodConfig, config.ContextOptionAgentInjectTimeout)

	// start the ssh session
	return StartSSHSession(
		ctx,
		"",
		cmd.Command,
		cmd.AgentForwarding,
		func(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			command := fmt.Sprintf("'%s' helper ssh-server --stdio", machineClient.AgentPath())
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
			},
				machineClient.AgentLocal(),
				machineClient.AgentPath(),
				machineClient.AgentURL(),
				true,
				command,
				stdin,
				stdout,
				stderr,
				log.Default.ErrorStreamOnly(),
				timeout)
		}, writer)
}

type ExecFunc func(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error

func StartSSHSession(ctx context.Context, user, command string, agentForwarding bool, exec ExecFunc, stderr io.Writer) error {
	// create readers
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	defer stdoutReader.Close()
	defer stdoutWriter.Close()
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	defer stdinWriter.Close()
	defer stdinReader.Close()

	// start ssh machine
	errChan := make(chan error, 1)
	go func() {
		errChan <- exec(ctx, stdinReader, stdoutWriter, stderr)
	}()

	sshClient, err := devssh.StdioClientWithUser(stdoutReader, stdinWriter, user, false)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	return RunSSHSession(ctx, sshClient, agentForwarding, command, stderr)
}

func RunSSHSession(ctx context.Context, sshClient *ssh.Client, agentForwarding bool, command string, stderr io.Writer) error {
	// create a new session
	session, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// request agent forwarding
	authSock := devsshagent.GetSSHAuthSocket()
	if agentForwarding && authSock != "" {
		err = devsshagent.ForwardToRemote(sshClient, authSock)
		if err != nil {
			return errors.Errorf("forward agent: %v", err)
		}

		err = devsshagent.RequestAgentForwarding(session)
		if err != nil {
			return errors.Errorf("request agent forwarding: %v", err)
		}
	}

	stdout := os.Stdout
	stdin := os.Stdin

	if isatty.IsTerminal(stdout.Fd()) {
		state, err := term.MakeRaw(int(stdout.Fd()))
		if err != nil {
			return err
		}
		defer func() {
			_ = term.Restore(int(stdout.Fd()), state)
		}()

		windowChange := devssh.WatchWindowSize(ctx)
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-windowChange:
				}
				width, height, err := term.GetSize(int(stdout.Fd()))
				if err != nil {
					continue
				}
				_ = session.WindowChange(height, width)
			}
		}()

		// get initial terminal
		t := "xterm-256color"
		termEnv, ok := os.LookupEnv("TERM")
		if ok {
			t = termEnv
		}
		// get initial window size
		width, height := 80, 40
		if w, h, err := term.GetSize(int(stdout.Fd())); err == nil {
			width, height = w, h
		}
		if err = session.RequestPty(t, height, width, ssh.TerminalModes{}); err != nil {
			return fmt.Errorf("request pty: %w", err)
		}
	}

	session.Stdin = stdin
	session.Stdout = stdout
	session.Stderr = stderr
	if command == "" {
		if err := session.Shell(); err != nil {
			return fmt.Errorf("start ssh session with shell: %w", err)
		}
	} else {
		if err := session.Start(command); err != nil {
			return fmt.Errorf("start ssh session with command %s: %w", command, err)
		}
	}

	if err := session.Wait(); err != nil {
		return fmt.Errorf("ssh session: %w", err)
	}

	return nil
}
