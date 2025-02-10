package huh

import "fmt"

// Option is an option for select fields.
type Option[T comparable] struct {
	Key      string
	Value    T
	selected bool
}

// NewOptions returns new options from a list of values.
func NewOptions[T comparable](values ...T) []Option[T] {
	options := make([]Option[T], len(values))
	for i, o := range values {
		options[i] = Option[T]{
			Key:   fmt.Sprint(o),
			Value: o,
		}
	}
	return options
}

// NewOption returns a new select option.
func NewOption[T comparable](key string, value T) Option[T] {
	return Option[T]{Key: key, Value: value}
}

// Selected sets whether the option is currently selected.
func (o Option[T]) Selected(selected bool) Option[T] {
	o.selected = selected
	return o
}

// String returns the key of the option.
func (o Option[T]) String() string {
	return o.Key
}
