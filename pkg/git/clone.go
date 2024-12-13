package git

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

type CloneStrategy string

const (
	FullCloneStrategy     CloneStrategy = ""
	BloblessCloneStrategy CloneStrategy = "blobless"
	TreelessCloneStrategy CloneStrategy = "treeless"
	ShallowCloneStrategy  CloneStrategy = "shallow"
	BareCloneStrategy     CloneStrategy = "bare"
)

type Cloner interface {
	Clone(ctx context.Context, repository string, targetDir string, extraArgs, extraEnv []string, log log.Logger) error
}

type Option func(*cloner)

func WithCloneStrategy(strategy CloneStrategy) Option {
	return func(c *cloner) {
		c.cloneStrategy = strategy
	}
}

func WithRecursiveSubmodules() Option {
	return func(c *cloner) {
		c.extraArgs = append(c.extraArgs, "--recurse-submodules")
	}
}

func NewClonerWithOpts(options ...Option) Cloner {
	cloner := &cloner{}
	for _, opt := range options {
		opt(cloner)
	}
	return cloner
}

func NewCloner(strategy CloneStrategy) Cloner {
	return NewClonerWithOpts(WithCloneStrategy(strategy))
}

var _ pflag.Value = (*CloneStrategy)(nil)

func (s *CloneStrategy) Set(v string) error {
	switch v {
	case string(FullCloneStrategy),
		string(BloblessCloneStrategy),
		string(TreelessCloneStrategy),
		string(ShallowCloneStrategy),
		string(BareCloneStrategy):
		{
			*s = CloneStrategy(v)
			return nil
		}
	default:
		return fmt.Errorf("CloneStrategy %s not supported", v)
	}
}

func (s *CloneStrategy) Type() string {
	return "cloneStrategy"
}

func (s *CloneStrategy) String() string {
	return string(*s)
}

type cloner struct {
	extraArgs     []string
	cloneStrategy CloneStrategy
}

var _ Cloner = &cloner{}

func (c *cloner) initialArgs() []string {
	switch c.cloneStrategy {
	case BloblessCloneStrategy:
		return []string{"clone", "--filter=blob:none"}
	case TreelessCloneStrategy:
		return []string{"clone", "--filter=tree:0"}
	case ShallowCloneStrategy:
		return []string{"clone", "--depth=1"}
	case BareCloneStrategy:
		return []string{"clone", "--bare", "--depth=1"}
	case FullCloneStrategy:
	default:
	}
	return []string{"clone"}
}

func (c *cloner) Clone(ctx context.Context, repository string, targetDir string, extraArgs, extraEnv []string, log log.Logger) error {
	args := c.initialArgs()
	args = append(args, extraArgs...)
	args = append(args, c.extraArgs...)
	args = append(args, repository, targetDir)
	return run(ctx, args, extraEnv, log)
}

func run(ctx context.Context, args []string, extraEnv []string, log log.Logger) error {
	var buf bytes.Buffer

	args = append(args, "--progress")

	gitCommand := CommandContext(ctx, args...)
	gitCommand.Stdout = &buf
	gitCommand.Stderr = &buf
	gitCommand.Env = append(gitCommand.Env, extraEnv...)

	// git always prints progress output to stderr,
	// we need to check the exit code to decide where the logs should go
	if err := gitCommand.Run(); err != nil {
		// report as error
		if _, err2 := io.Copy(log.Writer(logrus.ErrorLevel, false), &buf); err2 != nil {
			return err2
		}
		return err
	}

	// report as debug
	if _, err := io.Copy(log.Writer(logrus.DebugLevel, false), &buf); err != nil {
		return err
	}

	return nil
}
