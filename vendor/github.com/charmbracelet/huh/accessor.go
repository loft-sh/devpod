package huh

// Accessor give read/write access to field values.
type Accessor[T any] interface {
	Get() T
	Set(value T)
}

// EmbeddedAccessor is a basic accessor, acting as the default one for fields.
type EmbeddedAccessor[T any] struct {
	value T
}

// Get gets the value.
func (a *EmbeddedAccessor[T]) Get() T {
	return a.value
}

// Set sets the value.
func (a *EmbeddedAccessor[T]) Set(value T) {
	a.value = value
}

// PointerAccessor allows field value to be exposed as a pointed variable.
type PointerAccessor[T any] struct {
	value *T
}

// NewPointerAccessor returns a new pointer accessor.
func NewPointerAccessor[T any](value *T) *PointerAccessor[T] {
	return &PointerAccessor[T]{
		value: value,
	}
}

// Get gets the value.
func (a *PointerAccessor[T]) Get() T {
	return *a.value
}

// Set sets the value.
func (a *PointerAccessor[T]) Set(value T) {
	*a.value = value
}
