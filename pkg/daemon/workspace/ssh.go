package workspace

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

// RunSshServer starts the SSH server.
func RunSshServer(ctx context.Context, d *Daemon, errChan chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()
	binaryPath, err := os.Executable()
	if err != nil {
		errChan <- err
		return
	}

	args := []string{"agent", "container", "ssh-server"}
	if d.Config.Ssh.Workdir != "" {
		args = append(args, "--workdir", d.Config.Ssh.Workdir)
	}
	if d.Config.Ssh.User != "" {
		args = append(args, "--remote-user", d.Config.Ssh.User)
	}

	sshCmd := exec.Command(binaryPath, args...)
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	if err := sshCmd.Start(); err != nil {
		errChan <- fmt.Errorf("failed to start SSH server: %w", err)
		return
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			if sshCmd.Process != nil {
				if err := sshCmd.Process.Signal(syscall.SIGTERM); err != nil {
					errChan <- fmt.Errorf("failed to send SIGTERM to SSH server: %w", err)
				}
			}
		case <-done:
		}
	}()

	if err := sshCmd.Wait(); err != nil {
		errChan <- fmt.Errorf("SSH server exited abnormally: %w", err)
		close(done)
		return
	}
	close(done)
}
