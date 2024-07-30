// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

// Package syncs contains additional sync types and functionality.
package syncs

import (
	"context"
	"sync"
	"sync/atomic"

	"tailscale.com/util/mak"
)

// ClosedChan returns a channel that's already closed.
func ClosedChan() <-chan struct{} { return closedChan }

var closedChan = initClosedChan()

func initClosedChan() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

// AtomicValue is the generic version of atomic.Value.
type AtomicValue[T any] struct {
	v atomic.Value
}

// Load returns the value set by the most recent Store.
// It returns the zero value for T if the value is empty.
func (v *AtomicValue[T]) Load() T {
	x, _ := v.LoadOk()
	return x
}

// LoadOk is like Load but returns a boolean indicating whether the value was
// loaded.
func (v *AtomicValue[T]) LoadOk() (_ T, ok bool) {
	x := v.v.Load()
	if x != nil {
		return x.(T), true
	}
	var zero T
	return zero, false
}

// Store sets the value of the Value to x.
func (v *AtomicValue[T]) Store(x T) {
	v.v.Store(x)
}

// Swap stores new into Value and returns the previous value.
// It returns the zero value for T if the value is empty.
func (v *AtomicValue[T]) Swap(x T) (old T) {
	oldV := v.v.Swap(x)
	if oldV != nil {
		return oldV.(T)
	}
	return old
}

// CompareAndSwap executes the compare-and-swap operation for the Value.
func (v *AtomicValue[T]) CompareAndSwap(oldV, newV T) (swapped bool) {
	return v.v.CompareAndSwap(oldV, newV)
}

// WaitGroupChan is like a sync.WaitGroup, but has a chan that closes
// on completion that you can wait on. (This, you can only use the
// value once)
// Also, its zero value is not usable. Use the constructor.
type WaitGroupChan struct {
	n    int64         // atomic
	done chan struct{} // closed on transition to zero
}

// NewWaitGroupChan returns a new single-use WaitGroupChan.
func NewWaitGroupChan() *WaitGroupChan {
	return &WaitGroupChan{done: make(chan struct{})}
}

// DoneChan returns a channel that's closed on completion.
func (wg *WaitGroupChan) DoneChan() <-chan struct{} { return wg.done }

// Add adds delta, which may be negative, to the WaitGroupChan
// counter. If the counter becomes zero, all goroutines blocked on
// Wait or the Done chan are released. If the counter goes negative,
// Add panics.
//
// Note that calls with a positive delta that occur when the counter
// is zero must happen before a Wait. Calls with a negative delta, or
// calls with a positive delta that start when the counter is greater
// than zero, may happen at any time. Typically this means the calls
// to Add should execute before the statement creating the goroutine
// or other event to be waited for.
func (wg *WaitGroupChan) Add(delta int) {
	n := atomic.AddInt64(&wg.n, int64(delta))
	if n == 0 {
		close(wg.done)
	}
}

// Decr decrements the WaitGroup counter by one.
//
// (It is like sync.WaitGroup's Done method, but we don't use Done in
// this type, because it's ambiguous between Context.Done and
// WaitGroup.Done. So we use DoneChan and Decr instead.)
func (wg *WaitGroupChan) Decr() {
	wg.Add(-1)
}

// Wait blocks until the WaitGroupChan counter is zero.
func (wg *WaitGroupChan) Wait() { <-wg.done }

// Semaphore is a counting semaphore.
//
// Use NewSemaphore to create one.
type Semaphore struct {
	c chan struct{}
}

// NewSemaphore returns a semaphore with resource count n.
func NewSemaphore(n int) Semaphore {
	return Semaphore{c: make(chan struct{}, n)}
}

// Acquire blocks until a resource is acquired.
func (s Semaphore) Acquire() {
	s.c <- struct{}{}
}

// AcquireContext reports whether the resource was acquired before the ctx was done.
func (s Semaphore) AcquireContext(ctx context.Context) bool {
	select {
	case s.c <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

// TryAcquire reports, without blocking, whether the resource was acquired.
func (s Semaphore) TryAcquire() bool {
	select {
	case s.c <- struct{}{}:
		return true
	default:
		return false
	}
}

// Release releases a resource.
func (s Semaphore) Release() {
	<-s.c
}

// Map is a Go map protected by a [sync.RWMutex].
// It is preferred over [sync.Map] for maps with entries that change
// at a relatively high frequency.
// This must not be shallow copied.
type Map[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

// Load loads the value for the provided key and whether it was found.
func (m *Map[K, V]) Load(key K) (value V, loaded bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, loaded = m.m[key]
	return value, loaded
}

// LoadFunc calls f with the value for the provided key
// regardless of whether the entry exists or not.
// The lock is held for the duration of the call to f.
func (m *Map[K, V]) LoadFunc(key K, f func(value V, loaded bool)) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, loaded := m.m[key]
	f(value, loaded)
}

// Store stores the value for the provided key.
func (m *Map[K, V]) Store(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	mak.Set(&m.m, key, value)
}

// LoadOrStore returns the value for the given key if it exists
// otherwise it stores value.
func (m *Map[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	if actual, loaded = m.Load(key); loaded {
		return actual, loaded
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	actual, loaded = m.m[key]
	if !loaded {
		actual = value
		mak.Set(&m.m, key, value)
	}
	return actual, loaded
}

// LoadOrInit returns the value for the given key if it exists
// otherwise f is called to construct the value to be set.
// The lock is held for the duration to prevent duplicate initialization.
func (m *Map[K, V]) LoadOrInit(key K, f func() V) (actual V, loaded bool) {
	if actual, loaded := m.Load(key); loaded {
		return actual, loaded
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if actual, loaded = m.m[key]; loaded {
		return actual, loaded
	}

	loaded = false
	actual = f()
	mak.Set(&m.m, key, actual)
	return actual, loaded
}

// LoadAndDelete returns the value for the given key if it exists.
// It ensures that the map is cleared of any entry for the key.
func (m *Map[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, loaded = m.m[key]
	if loaded {
		delete(m.m, key)
	}
	return value, loaded
}

// Delete deletes the entry identified by key.
func (m *Map[K, V]) Delete(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.m, key)
}

// Range iterates over the map in undefined order calling f for each entry.
// Iteration stops if f returns false. Map changes are blocked during iteration.
func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for k, v := range m.m {
		if !f(k, v) {
			return
		}
	}
}

// Len returns the length of the map.
func (m *Map[K, V]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.m)
}

// Clear removes all entries from the map.
func (m *Map[K, V]) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	clear(m.m)
}

// WaitGroup is identical to [sync.WaitGroup],
// but provides a Go method to start a goroutine.
type WaitGroup struct{ sync.WaitGroup }

// Go calls the given function in a new goroutine.
// It automatically increments the counter before execution and
// automatically decrements the counter after execution.
// It must not be called concurrently with Wait.
func (wg *WaitGroup) Go(f func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		f()
	}()
}
