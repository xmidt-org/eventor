// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package eventor

import (
	"container/list"
	"sync"
)

// CancelFunc removes the listener it's associated with and cancels any
// future events sent to that listener.
//
// A CancelFunc is idempotent:  after the first invocation, calling this
// closure will have no effect.
type CancelFunc func()

// Eventor is a generic container of Eventor that is safe for concurrent access
// and concurrent dispatch of events through the visit method.
type Eventor[T any] struct {
	lock      sync.RWMutex
	listeners *list.List
}

// Cancel creates an idempotent closure that removes the given linked list element.
func (l *Eventor[T]) Cancel(e *list.Element) CancelFunc {
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
// The returned closure is idempotent:  after the first invocation, calling
// this closure will have no effect.
func (l *Eventor[T]) Add(listener T) CancelFunc {
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
