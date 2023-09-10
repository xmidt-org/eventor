// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package eventor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasicUseCase(t *testing.T) {
	assert := assert.New(t)

	// Define a listener type.  It can be anything.
	type listenerType int

	// Create an Eventor that holds listeners of type listenerType.
	eator := Eventor[*listenerType]{}

	// Show that nothing happens if we visit an empty Eventor.
	eator.Visit(func(i *listenerType) {
		assert.Fail("should not be called")
	})

	// Create 3 listeners.
	var a listenerType
	var b listenerType
	var c listenerType

	// Add two of them to the Eventor.
	cancelA := eator.Add(&a)
	cancelB := eator.Add(&b)

	// Show that we can visit the Eventor and the two listeners we added are visited.
	var count int
	eator.Visit(func(i *listenerType) {
		count++
	})
	assert.Equal(2, count)

	// Show that we can add a nil listener and that it is visited.
	cancelD := eator.Add(nil)
	count = 0
	eator.Visit(func(i *listenerType) {
		count++
	})
	assert.Equal(3, count)

	// Show it's safe to call cancel on a nil listener.
	cancelD()

	// Let's add the third listener.
	// Show that we can visit the Eventor and all three listeners are visited.
	cancelC := eator.Add(&c)
	count = 0
	eator.Visit(func(i *listenerType) {
		count++
	})
	assert.Equal(3, count)

	// Show that we can cancel a listener and it is no longer visited.
	cancelB()
	count = 0
	eator.Visit(func(i *listenerType) {
		count++
	})
	assert.Equal(2, count)

	// Oops, we passed in a nil function.  Show that it is safe.
	eator.Visit(nil)

	// Show that we can cancel a listener and it is no longer visited.
	cancelA()
	cancelC()
	eator.Visit(func(i *listenerType) {
		assert.Fail("should not be called")
	})

	// Show that cancel is idempotent.
	cancelA()
	cancelB()
	cancelC()
}
