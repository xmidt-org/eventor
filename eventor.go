// SPDX-FileCopyrightText: 2023-2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

// Package eventor provides Eventor, a generic, concurrency-safe set of
// listeners that events can be dispatched to.
package eventor

import (
	"container/list"
	"sync"
)

// Eventor is a generic container of Eventor that is safe for concurrent access
// and concurrent dispatch of events through the visit method.
type Eventor[T any] struct {
	lock      sync.RWMutex
	listeners *list.List
}

// Cancel creates an idempotent closure that removes the given linked list element.
//
// The returned cancel func removes the listener it's associated with and
// cancels any future events sent to that listener.  It is safe to call the
// cancel func multiple times, as it is idempotent and additional calls will
// have no effect.
func (l *Eventor[T]) Cancel(e *list.Element) (cancel func()) {
	return func() {
		l.lock.Lock()
		defer l.lock.Unlock()

		// NOTE: Remove is idempotent: it will not do anything if e is not in the list
		l.listeners.Remove(e)
	}
}

// Add inserts a new listener into the list and returns a closure that will
// remove the listener from the list.
//
// This method is atomic with respect to Visit.
//
// The returned cancel func removes the listener it's associated with and
// cancels any future events sent to that listener.  It is safe to call the
// cancel func multiple times, as it is idempotent and additional calls will
// have no effect.
func (l *Eventor[T]) Add(listener T) (cancel func()) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if l.listeners == nil {
		l.listeners = list.New()
	}

	e := l.listeners.PushBack(listener)
	return l.Cancel(e)
}

// Visit applies the given closure to each listener in the list.
//
// The listeners are snapshotted at the start of the call, and f is called
// without any lock held.  As a result:
//
//   - A listener added during a visit is not called in that visit.
//   - A listener canceled during a visit may still be called once in that
//     visit if it had not been reached yet.
//   - Listeners may safely call Add, a cancel func, or Visit from inside f.
//   - A slow f does not block Add or cancel on other goroutines.
func (l *Eventor[T]) Visit(f func(T)) {
	if f == nil {
		return
	}

	for _, listener := range l.snapshot() {
		f(listener)
	}
}

// snapshot returns a copy of the current listeners, or nil if there are none.
func (l *Eventor[T]) snapshot() []T {
	l.lock.RLock()
	defer l.lock.RUnlock()

	if l.listeners == nil || l.listeners.Len() == 0 {
		return nil
	}

	listeners := make([]T, 0, l.listeners.Len())
	for e := l.listeners.Front(); e != nil; e = e.Next() {
		listeners = append(listeners, e.Value.(T))
	}

	return listeners
}

// Len returns the number of listeners in the list.
func (l *Eventor[T]) Len() int {
	l.lock.RLock()
	defer l.lock.RUnlock()

	if l.listeners == nil {
		return 0
	}

	return l.listeners.Len()
}
