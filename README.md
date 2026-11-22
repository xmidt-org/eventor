# eventor
A simple library to make the observer pattern simpler to implement.

[![Build Status](https://github.com/xmidt-org/eventor/actions/workflows/ci.yml/badge.svg)](https://github.com/xmidt-org/eventor/actions/workflows/ci.yml)
[![codecov.io](https://codecov.io/gh/xmidt-org/eventor/branch/main/graph/badge.svg)](https://codecov.io/gh/xmidt-org/eventor)
[![Apache V2 License](https://img.shields.io/badge/license-Apache%20V2-blue.svg)](https://github.com/xmidt-org/eventor/blob/main/LICENSE)
[![GitHub Release](https://img.shields.io/github/release/xmidt-org/eventor.svg)](https://github.com/xmidt-org/eventor/releases)
[![GoDoc](https://pkg.go.dev/badge/github.com/xmidt-org/eventor)](https://pkg.go.dev/github.com/xmidt-org/eventor)

## Summary

`eventor` provides `Eventor[T]`, a generic, concurrency-safe set of listeners
that you can add to, cancel from, and dispatch events to. The zero value is
ready to use.

## Install

```sh
go get github.com/xmidt-org/eventor
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/xmidt-org/eventor"
)

// A listener can be any type; a func is the most common choice.
type Listener func(msg string)

func main() {
	var events eventor.Eventor[Listener]

	cancel := events.Add(func(msg string) {
		fmt.Println("got:", msg)
	})
	defer cancel()

	// Dispatch an event to every listener.
	events.Visit(func(l Listener) {
		l("hello")
	})
}
```

`Add` returns a cancel func that removes the listener. It is idempotent, so
calling it more than once is safe.

## Dispatch semantics

`Visit` takes a snapshot of the listeners and then calls each one without
holding any lock. This means:

- Listeners may call `Add`, a cancel func, or `Visit` from inside the
  callback, so a one-shot listener can cancel itself.
- A listener added during a visit is first called on the next visit.
- A listener canceled during a visit may still be called once in that visit
  if it had not been reached yet.
- A slow listener does not block `Add` or cancel on other goroutines, but it
  does delay the remaining listeners in that visit.

## Code of Conduct

This project and everyone participating in it are governed by the
[XMiDT Code Of Conduct](https://xmidt.io/docs/community/code_of_conduct/).
By participating, you agree to this Code.

## Contributing

Refer to [CONTRIBUTING.md](CONTRIBUTING.md).
