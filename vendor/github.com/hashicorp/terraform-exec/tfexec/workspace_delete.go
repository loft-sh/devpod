package tfexec

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
)

type workspaceDeleteConfig struct {
	lock        bool
	lockTimeout string
	force       bool
}

var defaultWorkspaceDeleteOptions = workspaceDeleteConfig{
	lock:        true,
	lockTimeout: "0s",
}

// WorkspaceDeleteCmdOption represents options that are applicable to the WorkspaceDelete method.
type WorkspaceDeleteCmdOption interface {
	configureWorkspaceDelete(*workspaceDeleteConfig)
}

func (opt *LockOption) configureWorkspaceDelete(conf *workspaceDeleteConfig) {
	conf.lock = opt.lock
}

func (opt *LockTimeoutOption) configureWorkspaceDelete(conf *workspaceDeleteConfig) {
	conf.lockTimeout = opt.timeout
}

func (opt *ForceOption) configureWorkspaceDelete(conf *workspaceDeleteConfig) {
	conf.force = opt.force
}

// WorkspaceDelete represents the workspace delete subcommand to the Terraform CLI.
func (tf *Terraform) WorkspaceDelete(ctx context.Context, workspace string, opts ...WorkspaceDeleteCmdOption) error {
	cmd, err := tf.workspaceDeleteCmd(ctx, workspace, opts...)
	if err != nil {
		return err
	}
	return tf.runTerraformCmd(ctx, cmd)
}

func (tf *Terraform) workspaceDeleteCmd(ctx context.Context, workspace string, opts ...WorkspaceDeleteCmdOption) (*exec.Cmd, error) {
	c := defaultWorkspaceDeleteOptions

	for _, o := range opts {
		switch o.(type) {
		case *LockOption, *LockTimeoutOption:
			err := tf.compatible(ctx, tf0_12_0, nil)
			if err != nil {
				return nil, fmt.Errorf("-lock and -lock-timeout were added to workspace delete in Terraform 0.12: %w", err)
			}
		}

		o.configureWorkspaceDelete(&c)
	}

	args := []string{"workspace", "delete", "-no-color"}

	if c.force {
		args = append(args, "-force")
	}
	if c.lockTimeout != "" && c.lockTimeout != defaultWorkspaceDeleteOptions.lockTimeout {
		// only pass if not default, so we don't need to worry about the 0.11 version check
		args = append(args, "-lock-timeout="+c.lockTimeout)
	}
	if !c.lock {
		// only pass if false, so we don't need to worry about the 0.11 version check
		args = append(args, "-lock="+strconv.FormatBool(c.lock))
	}

	args = append(args, workspace)

	cmd := tf.buildTerraformCmd(ctx, nil, args...)

	return cmd, nil
}
