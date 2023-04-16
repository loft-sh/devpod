package git

import (
	"context"
	"os"
	"os/exec"
)

func CommandContext(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "GIT_TERMINAL_PROMPT=0")
	cmd.Env = append(cmd.Env, "GIT_SSH_COMMAND=ssh -oBatchMode=yes -oStrictHostKeyChecking=no")
	return cmd
}
