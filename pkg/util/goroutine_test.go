package util

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestSafelyGo(t *testing.T) {
	t.Run("normal execution", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		executed := false

		SafelyGo(func() {
			executed = true
			wg.Done()
		})

		wg.Wait()
		if !executed {
			t.Error("Function was not executed")
		}
	})

	t.Run("panic recovery", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		panicHandled := false

		SafelyGo(
			func() {
				defer wg.Done()
				panic("test panic")
			},
			func(r interface{}) {
				panicHandled = true
				if r != "test panic" {
					t.Errorf("Expected panic value 'test panic', got %v", r)
				}
			},
		)

		wg.Wait()
		// Give some time for the panic handler to execute
		time.Sleep(10 * time.Millisecond)
		if !panicHandled {
			t.Error("Panic was not handled")
		}
	})

	t.Run("panic recovery without handler", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)

		// This should not crash the test
		SafelyGo(func() {
			defer wg.Done()
			panic("test panic without handler")
		})

		wg.Wait()
		// Give some time for the panic to be logged
		time.Sleep(10 * time.Millisecond)
	})
}

func TestSafelyGoWithError(t *testing.T) {
	t.Run("function returns error", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)

		SafelyGoWithError(func() error {
			defer wg.Done()
			return errors.New("test error")
		})

		wg.Wait()
		// Give some time for the error to be logged
		time.Sleep(10 * time.Millisecond)
	})

	t.Run("function returns nil", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		executed := false

		SafelyGoWithError(func() error {
			executed = true
			wg.Done()
			return nil
		})

		wg.Wait()
		if !executed {
			t.Error("Function was not executed")
		}
	})

	t.Run("function panics", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		panicHandled := false

		SafelyGoWithError(
			func() error {
				defer wg.Done()
				panic("test panic in error function")
			},
			func(r interface{}) {
				panicHandled = true
			},
		)

		wg.Wait()
		time.Sleep(10 * time.Millisecond)
		if !panicHandled {
			t.Error("Panic was not handled")
		}
	})
}

func TestSafelyGoWithErrorHandler(t *testing.T) {
	t.Run("custom error handler", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		errorHandled := false

		SafelyGoWithErrorHandler(
			func() error {
				defer wg.Done()
				return errors.New("test error")
			},
			func(err error) {
				errorHandled = true
				if err.Error() != "test error" {
					t.Errorf("Expected error 'test error', got %v", err)
				}
			},
		)

		wg.Wait()
		time.Sleep(10 * time.Millisecond)
		if !errorHandled {
			t.Error("Error was not handled")
		}
	})

	t.Run("nil error handler uses default logging", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)

		SafelyGoWithErrorHandler(
			func() error {
				defer wg.Done()
				return errors.New("test error")
			},
			nil, // Should use default logging
		)

		wg.Wait()
		time.Sleep(10 * time.Millisecond)
	})
}

func TestMustSafelyGo(t *testing.T) {
	t.Run("nil function panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for nil function")
			}
		}()

		MustGo(nil)
	})

	t.Run("valid function executes", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		executed := false

		MustGo(func() {
			executed = true
			wg.Done()
		})

		wg.Wait()
		if !executed {
			t.Error("Function was not executed")
		}
	})
}

// Benchmark tests
func BenchmarkSafelyGo(b *testing.B) {
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		SafelyGo(func() {
			wg.Done()
		})
	}
	wg.Wait()
}

func BenchmarkSafelyGoWithPanic(b *testing.B) {
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		SafelyGo(func() {
			defer wg.Done()
			panic("benchmark panic")
		})
	}
	wg.Wait()
}
