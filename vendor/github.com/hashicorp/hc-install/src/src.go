package src

import (
	"context"
	"log"

	isrc "github.com/hashicorp/hc-install/internal/src"
)

// Source represents an installer, finder, or builder
type Source interface {
	IsSourceImpl() isrc.InstallSrcSigil
}

type Installable interface {
	Source
	Install(ctx context.Context) (string, error)
}

type Findable interface {
	Source
	Find(ctx context.Context) (string, error)
}

type Buildable interface {
	Source
	Build(ctx context.Context) (string, error)
}

type Validatable interface {
	Source
	Validate() error
}

type Removable interface {
	Source
	Remove(ctx context.Context) error
}

type LoggerSettable interface {
	SetLogger(logger *log.Logger)
}
