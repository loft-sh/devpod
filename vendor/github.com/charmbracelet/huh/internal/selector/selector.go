package selector

// Selector is a helper type for selecting items.
type Selector[T any] struct {
	items []T
	index int
}

// NewSelector creates a new item selector.
func NewSelector[T any](items []T) *Selector[T] {
	return &Selector[T]{
		items: items,
	}
}

// Append adds an item to the selector.
func (s *Selector[T]) Append(item T) {
	s.items = append(s.items, item)
}

// Next moves the selector to the next item.
func (s *Selector[T]) Next() {
	if s.index < len(s.items)-1 {
		s.index++
	}
}

// Prev moves the selector to the previous item.
func (s *Selector[T]) Prev() {
	if s.index > 0 {
		s.index--
	}
}

// OnFirst returns true if the selector is on the first item.
func (s *Selector[T]) OnFirst() bool {
	return s.index == 0
}

// OnLast returns true if the selector is on the last item.
func (s *Selector[T]) OnLast() bool {
	return s.index == len(s.items)-1
}

// Selected returns the index of the current selected item.
func (s *Selector[T]) Selected() T {
	return s.items[s.index]
}

// Index returns the index of the current selected item.
func (s *Selector[T]) Index() int {
	return s.index
}

// Totoal returns the total number of items.
func (s *Selector[T]) Total() int {
	return len(s.items)
}

// SetIndex sets the selected item.
func (s *Selector[T]) SetIndex(i int) {
	if i < 0 || i >= len(s.items) {
		return
	}
	s.index = i
}

// Get returns the item at the given index.
func (s *Selector[T]) Get(i int) T {
	return s.items[i]
}

// Set sets the item at the given index.
func (s *Selector[T]) Set(i int, item T) {
	s.items[i] = item
}

// Range iterates over the items.
// The callback function should return true to continue the iteration.
func (s *Selector[T]) Range(f func(i int, item T) bool) {
	for i, item := range s.items {
		if !f(i, item) {
			break
		}
	}
}

// ReverseRange iterates over the items in reverse.
// The callback function should return true to continue the iteration.
func (s *Selector[T]) ReverseRange(f func(i int, item T) bool) {
	for i := len(s.items) - 1; i >= 0; i-- {
		if !f(i, s.items[i]) {
			break
		}
	}
}
