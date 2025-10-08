// Package util provides utility functions for the poorman-faas project.
package util

import (
	"fmt"
	"log"
	"runtime/debug"
)

// SafelyGo executes a function in a goroutine with panic recovery.
// If the function panics, the panic is recovered and logged.
// Optionally, a callback function can be provided to handle the recovered panic.
func SafelyGo(fn func(), onPanic ...func(interface{})) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic with stack trace
				log.Printf("Panic recovered in goroutine: %v\nStack trace:\n%s", r, debug.Stack())

				// Call the panic handler if provided
				if len(onPanic) > 0 && onPanic[0] != nil {
					onPanic[0](r)
				}
			}
		}()

		fn()
	}()
}

// SafelyGoWithError is similar to SafelyGo but for functions that return an error.
// If the function returns an error, it will be logged.
// If the function panics, the panic is recovered and logged.
func SafelyGoWithError(fn func() error, onPanic ...func(interface{})) {
	SafelyGo(func() {
		if err := fn(); err != nil {
			log.Printf("Error in goroutine: %v", err)
		}
	}, onPanic...)
}

// SafelyGoWithErrorHandler is similar to SafelyGoWithError but allows custom error handling.
func SafelyGoWithErrorHandler(fn func() error, onError func(error), onPanic ...func(interface{})) {
	SafelyGo(func() {
		if err := fn(); err != nil {
			if onError != nil {
				onError(err)
			} else {
				log.Printf("Error in goroutine: %v", err)
			}
		}
	}, onPanic...)
}

// MustGo is like SafelyGo but panics if fn is nil.
func MustGo(fn func(), onPanic ...func(interface{})) {
	if fn == nil {
		panic(fmt.Errorf("MustGo: fn cannot be nil"))
	}
	SafelyGo(fn, onPanic...)
}
