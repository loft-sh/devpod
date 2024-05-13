package git

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/pflag"
)

type CloneStrategy string

const (
	FullCloneStrategy     CloneStrategy = ""
	BloblessCloneStrategy CloneStrategy = "blobless"
	TreelessCloneStrategy CloneStrategy = "treeless"
	ShallowCloneStrategy  CloneStrategy = "shallow"
)

type Cloner interface {
	Clone(ctx context.Context, repository string, targetDir string, extraArgs []string, stdout, stderr io.Writer) error
}

func NewCloner(strategy CloneStrategy) Cloner {
	switch strategy {
	case BloblessCloneStrategy:
		return &bloblessClone{}
	case TreelessCloneStrategy:
		return &treelessClone{}
	case ShallowCloneStrategy:
		return &shallowClone{}
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
		string(ShallowCloneStrategy):
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

func (c *fullClone) Clone(ctx context.Context, repository string, targetDir string, extraArgs []string, stdout, stderr io.Writer) error {
	args := []string{"clone"}
	args = append(args, extraArgs...)
	args = append(args, repository, targetDir)
	return run(ctx, args, stdout, stderr)
}

type bloblessClone struct{}

var _ Cloner = &bloblessClone{}

func (c *bloblessClone) Clone(ctx context.Context, repository string, targetDir string, extraArgs []string, stdout, stderr io.Writer) error {
	args := []string{"clone", "--filter=blob:none"}
	args = append(args, extraArgs...)
	args = append(args, repository, targetDir)
	return run(ctx, args, stdout, stderr)
}

type treelessClone struct{}

var _ Cloner = treelessClone{}

func (c treelessClone) Clone(ctx context.Context, repository string, targetDir string, extraArgs []string, stdout, stderr io.Writer) error {
	args := []string{"clone", "--filter=tree:0"}
	args = append(args, extraArgs...)
	args = append(args, repository, targetDir)
	return run(ctx, args, stdout, stderr)
}

type shallowClone struct{}

var _ Cloner = shallowClone{}

func (c shallowClone) Clone(ctx context.Context, repository string, targetDir string, extraArgs []string, stdout, stderr io.Writer) error {
	args := []string{"clone", "--depth=1"}
	args = append(args, extraArgs...)
	args = append(args, repository, targetDir)
	return run(ctx, args, stdout, stderr)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	gitCommand := CommandContext(ctx, args...)
	gitCommand.Stdout = stdout
	gitCommand.Stderr = stderr
	return gitCommand.Run()
}
