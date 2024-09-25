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

func NewCloner(strategy CloneStrategy) Cloner {
	switch strategy {
	case BloblessCloneStrategy:
		return &bloblessClone{}
	case TreelessCloneStrategy:
		return &treelessClone{}
	case ShallowCloneStrategy:
		return &shallowClone{}
	case BareCloneStrategy:
		return &bareClone{}
	case FullCloneStrategy:
		return &fullClone{}
	default:
		return &fullClone{}
	}
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

type fullClone struct{}

var _ Cloner = &fullClone{}

func (c *fullClone) Clone(ctx context.Context, repository string, targetDir string, extraArgs, extraEnv []string, log log.Logger) error {
	args := []string{"clone"}
	args = append(args, extraArgs...)
	args = append(args, repository, targetDir)
	return run(ctx, args, extraEnv, log)
}

type bloblessClone struct{}

var _ Cloner = &bloblessClone{}

func (c *bloblessClone) Clone(ctx context.Context, repository string, targetDir string, extraArgs, extraEnv []string, log log.Logger) error {
	args := []string{"clone", "--filter=blob:none"}
	args = append(args, extraArgs...)
	args = append(args, repository, targetDir)
	return run(ctx, args, extraEnv, log)
}

type treelessClone struct{}

var _ Cloner = treelessClone{}

func (c treelessClone) Clone(ctx context.Context, repository string, targetDir string, extraArgs, extraEnv []string, log log.Logger) error {
	args := []string{"clone", "--filter=tree:0"}
	args = append(args, extraArgs...)
	args = append(args, repository, targetDir)
	return run(ctx, args, extraEnv, log)
}

type shallowClone struct{}

var _ Cloner = shallowClone{}

func (c shallowClone) Clone(ctx context.Context, repository string, targetDir string, extraArgs, extraEnv []string, log log.Logger) error {
	args := []string{"clone", "--depth=1"}
	args = append(args, extraArgs...)
	args = append(args, repository, targetDir)
	return run(ctx, args, extraEnv, log)
}

type bareClone struct{}

var _ Cloner = bareClone{}

func (c bareClone) Clone(ctx context.Context, repository string, targetDir string, extraArgs, extraEnv []string, log log.Logger) error {
	args := []string{"clone", "bare", "--depth=1"}
	args = append(args, extraArgs...)
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
