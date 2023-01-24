package product

import (
	"context"
	"time"

	"github.com/hashicorp/go-version"
)

type Product struct {
	// Name which identifies the product
	// on releases.hashicorp.com and in Checkpoint
	Name string

	// BinaryName represents name of the unpacked binary to be executed or built
	BinaryName BinaryNameFunc

	// GetVersion represents how to obtain the version of the product
	// reflecting any output or CLI flag differences
	GetVersion func(ctx context.Context, execPath string) (*version.Version, error)

	// BuildInstructions represents how to build the product "from scratch"
	BuildInstructions *BuildInstructions
}

type BinaryNameFunc func() string

type BuildInstructions struct {
	GitRepoURL string

	// CloneTimeout overrides default timeout
	// for cloning the repository
	CloneTimeout time.Duration

	// PreCloneCheck represents any checks to run
	// prior to building, such as verifying build
	// dependencies (e.g. whether Go is installed)
	PreCloneCheck Checker

	// PreCloneCheckTimeout overrides default timeout
	// for the PreCloneCheck
	PreCloneCheckTimeout time.Duration

	// Build represents how to build the product
	// after checking out the source code
	Build Builder

	// BuildTimeout overrides default timeout
	// for the Builder
	BuildTimeout time.Duration
}

type Checker interface {
	Check(ctx context.Context) error
}

type Builder interface {
	Build(ctx context.Context, repoDir, targetDir, binaryName string) (string, error)
	Remove(ctx context.Context) error
}
