// SPDX-FileCopyrightText: 2023-2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

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

// Visit applies the given closure to each listener in the list.  This method
// is atomic with respect to Add.
func (l *Eventor[T]) Visit(f func(T)) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	if l.listeners == nil || f == nil {
		return
	}

	for e := l.listeners.Front(); e != nil; e = e.Next() {
		f(e.Value.(T))
	}
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
