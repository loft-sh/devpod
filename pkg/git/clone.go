package git

import (
	"context"
	"fmt"

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
		if strategy == "" {
			strategy = FullCloneStrategy
		}
		c.cloneStrategy = strategy
	}
}

func WithRecursiveSubmodules() Option {
	return func(c *cloner) {
		c.extraArgs = append(c.extraArgs, "--recurse-submodules")
	}
}

func WithSkipLFS() Option {
	return func(c *cloner) {
		c.skipLFS = true
	}
}

func NewClonerWithOpts(options ...Option) Cloner {
	cloner := &cloner{
		cloneStrategy: FullCloneStrategy,
	}
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
	skipLFS       bool
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

type progressWriter struct {
	level logrus.Level
	log   log.Logger
}

func (w *progressWriter) Write(p []byte) (n int, err error) {
	return w.log.WriteLevel(w.level, p)
}

func (c *cloner) Clone(ctx context.Context, repository string, targetDir string, extraArgs, extraEnv []string, log log.Logger) error {
	args := c.initialArgs()
	args = append(args, extraArgs...)
	args = append(args, c.extraArgs...)
	args = append(args, repository, targetDir)
	args = append(args, "--progress")

	if c.skipLFS {
		extraEnv = append(extraEnv, "GIT_LFS_SKIP_SMUDGE=1")
	}

	w := &progressWriter{log: log, level: logrus.InfoLevel}
	gitCommand := CommandContext(ctx, extraEnv, args...)
	gitCommand.Stdout = w
	gitCommand.Stderr = w

	return gitCommand.Run()
}
