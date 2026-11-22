// SPDX-FileCopyrightText: 2023-2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package eventor

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBasicUseCase(t *testing.T) {
	assert := assert.New(t)

	// Define a listener type.  It can be anything.
	type listenerType int

	// Create an Eventor that holds listeners of type listenerType.
	eator := Eventor[*listenerType]{}

	// Show that nothing happens if we visit an empty Eventor.
	eator.Visit(func(_ *listenerType) {
		assert.Fail("should not be called")
	})

	// Create 3 listeners.
	var a listenerType
	var b listenerType
	var c listenerType

	// Add two of them to the Eventor.
	assert.Equal(0, eator.Len())
	cancelA := eator.Add(&a)
	assert.Equal(1, eator.Len())
	cancelB := eator.Add(&b)
	assert.Equal(2, eator.Len())

	// Show that we can visit the Eventor and the two listeners we added are visited.
	var count int
	eator.Visit(func(_ *listenerType) {
		count++
	})
	assert.Equal(2, count)

	// Show that we can add a nil listener and that it is visited.
	cancelD := eator.Add(nil)
	count = 0
	eator.Visit(func(_ *listenerType) {
		count++
	})
	assert.Equal(3, count)

	// Show it's safe to call cancel on a nil listener.
	cancelD()

	// Let's add the third listener.
	// Show that we can visit the Eventor and all three listeners are visited.
	cancelC := eator.Add(&c)
	count = 0
	eator.Visit(func(_ *listenerType) {
		count++
	})
	assert.Equal(3, count)

	// Show that we can cancel a listener and it is no longer visited.
	cancelB()
	count = 0
	eator.Visit(func(_ *listenerType) {
		count++
	})
	assert.Equal(2, count)

	// Oops, we passed in a nil function.  Show that it is safe.
	eator.Visit(nil)

	// Show that we can cancel a listener and it is no longer visited.
	cancelA()
	cancelC()
	eator.Visit(func(_ *listenerType) {
		assert.Fail("should not be called")
	})

	// Show that cancel is idempotent.
	cancelA()
	cancelB()
	cancelC()
}

// runWithTimeout runs f in a goroutine and fails the test if it doesn't finish
// within a second, so a deadlock regression fails fast instead of hanging CI.
func runWithTimeout(t *testing.T, f func()) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		defer close(done)
		f()
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out: probable deadlock")
	}
}

func callIt(fn func()) { fn() }

func TestVisitListenerCancelsSelf(t *testing.T) {
	assert := assert.New(t)

	var eator Eventor[func()]

	var calls int
	var cancel func()
	cancel = eator.Add(func() {
		calls++
		cancel()
	})
	eator.Add(func() {})
	assert.Equal(2, eator.Len())

	runWithTimeout(t, func() {
		eator.Visit(callIt)
		eator.Visit(callIt)
	})

	assert.Equal(1, calls)
	assert.Equal(1, eator.Len())
}

func TestVisitListenerAddsListener(t *testing.T) {
	assert := assert.New(t)

	var eator Eventor[func()]

	var added bool
	var newCalls int
	eator.Add(func() {
		if !added {
			added = true
			eator.Add(func() { newCalls++ })
		}
	})

	runWithTimeout(t, func() { eator.Visit(callIt) })
	assert.Equal(0, newCalls, "listener added during a visit must not be called in that visit")
	assert.Equal(2, eator.Len())

	runWithTimeout(t, func() { eator.Visit(callIt) })
	assert.Equal(1, newCalls, "listener added during a visit must be called in the next visit")
}

func TestVisitNestedWithPendingWriter(t *testing.T) {
	assert := assert.New(t)

	var eator Eventor[func()]

	var nested bool
	var nestedCalls int
	addDone := make(chan struct{})
	eator.Add(func() {
		if nested {
			nestedCalls++
			return
		}
		nested = true

		// Get a writer queued on the lock, then visit again from inside the
		// callback. With a lock held across callbacks, the nested read lock
		// waits behind the writer and the writer waits behind the outer read.
		go func() {
			defer close(addDone)
			eator.Add(func() {})
		}()
		time.Sleep(10 * time.Millisecond)

		eator.Visit(callIt)
	})

	runWithTimeout(t, func() {
		eator.Visit(callIt)
		<-addDone
	})

	assert.Equal(1, nestedCalls)
	assert.Equal(2, eator.Len())
}

func TestVisitCancelOtherDuringVisit(t *testing.T) {
	assert := assert.New(t)

	var eator Eventor[func()]

	var calls2 int
	var cancel2 func()
	eator.Add(func() { cancel2() })
	cancel2 = eator.Add(func() { calls2++ })

	// Visit snapshots the listeners, so listener 2 is still called in the
	// pass in which it was canceled...
	runWithTimeout(t, func() { eator.Visit(callIt) })
	assert.Equal(1, calls2)
	assert.Equal(1, eator.Len())

	// ...but not in any later pass.
	runWithTimeout(t, func() { eator.Visit(callIt) })
	assert.Equal(1, calls2)
}

func TestConcurrentAddCancelVisit(t *testing.T) {
	const (
		writers    = 8
		visitors   = 8
		iterations = 200
	)

	var eator Eventor[func()]
	var wg sync.WaitGroup

	runWithTimeout(t, func() {
		for i := 0; i < writers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					cancel := eator.Add(func() {})
					cancel()
				}
			}()
		}

		for i := 0; i < visitors; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					eator.Visit(callIt)
					_ = eator.Len()
				}
			}()
		}

		wg.Wait()
	})

	assert.Equal(t, 0, eator.Len())
}
