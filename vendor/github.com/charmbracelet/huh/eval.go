package huh

import (
	"time"

	"github.com/mitchellh/hashstructure/v2"
)

// Eval is an evaluatable value, it stores a cached value and a function to
// recompute it. It's bindings are what we check to see if we need to recompute
// the value.
//
// By default it is also cached.
type Eval[T any] struct {
	val T
	fn  func() T

	bindings     any
	bindingsHash uint64
	cache        map[uint64]T

	loading      bool
	loadingStart time.Time
}

const spinnerShowThreshold = 25 * time.Millisecond

func hash(val any) uint64 {
	hash, _ := hashstructure.Hash(val, hashstructure.FormatV2, nil)
	return hash
}

func (e *Eval[T]) shouldUpdate() (bool, uint64) {
	if e.fn == nil {
		return false, 0
	}
	newHash := hash(e.bindings)
	return e.bindingsHash != newHash, newHash
}

func (e *Eval[T]) loadFromCache() bool {
	val, ok := e.cache[e.bindingsHash]
	if ok {
		e.loading = false
		e.val = val
	}
	return ok
}

func (e *Eval[T]) update(val T) {
	e.val = val
	e.cache[e.bindingsHash] = val
	e.loading = false
}

type updateTitleMsg struct {
	id    int
	hash  uint64
	title string
}

type updateDescriptionMsg struct {
	id          int
	hash        uint64
	description string
}

type updatePlaceholderMsg struct {
	id          int
	hash        uint64
	placeholder string
}

type updateSuggestionsMsg struct {
	id          int
	hash        uint64
	suggestions []string
}

type updateOptionsMsg[T comparable] struct {
	id      int
	hash    uint64
	options []Option[T]
}
