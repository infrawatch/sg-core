package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type metric struct {
	expired bool
	delete  func()
	deleted bool
}

func (m *metric) Expired(i time.Duration) bool {
	return m.expired
}

func (m *metric) Delete() bool {
	if m.delete != nil {
		m.delete()
	}
	return m.deleted
}

func TestExpiry(t *testing.T) {
	ep := newExpiryProc(1)

	t.Run("single entry", func(t *testing.T) {
		deleted := false
		ep.register(&metric{
			expired: true,
			delete: func() {
				deleted = true
			},
			deleted: true,
		})
		assert.Equal(t, 1, ep.entries.Len(), "entry not registered")
		ep.check()
		assert.Equal(t, true, deleted, "expiry.delete() not called")
		assert.Equal(t, 0, ep.entries.Len(), "entry not removed after expiration")
	})

	t.Run("multiple entries", func(t *testing.T) {
		ep := newExpiryProc(1)
		deleteCount := 0

		// Register 3 expired entries
		for i := 0; i < 3; i++ {
			ep.register(&metric{
				expired: true,
				delete: func() {
					deleteCount++
				},
				deleted: true,
			})
		}

		assert.Equal(t, 3, ep.entries.Len(), "entries not registered")
		ep.check()
		assert.Equal(t, 3, deleteCount, "not all delete() called")
		assert.Equal(t, 0, ep.entries.Len(), "entries not removed after expiration")
	})

	t.Run("entry not expired", func(t *testing.T) {
		ep := newExpiryProc(1)
		deleted := false

		ep.register(&metric{
			expired: false,
			delete: func() {
				deleted = true
			},
			deleted: true,
		})

		assert.Equal(t, 1, ep.entries.Len(), "entry not registered")
		ep.check()
		assert.Equal(t, false, deleted, "delete() should not be called for non-expired entry")
		assert.Equal(t, 1, ep.entries.Len(), "non-expired entry should remain in list")
	})

	t.Run("entry expired but delete returns false", func(t *testing.T) {
		ep := newExpiryProc(1)
		deleted := false

		ep.register(&metric{
			expired: true,
			delete: func() {
				deleted = true
			},
			deleted: false, // Delete returns false
		})

		assert.Equal(t, 1, ep.entries.Len(), "entry not registered")
		ep.check()
		assert.Equal(t, true, deleted, "delete() should be called")
		assert.Equal(t, 1, ep.entries.Len(), "entry should remain if Delete() returns false")
	})

	t.Run("mixed expired and non-expired entries", func(t *testing.T) {
		ep := newExpiryProc(1)
		deleteCount := 0

		// Register expired entry
		ep.register(&metric{
			expired: true,
			delete: func() {
				deleteCount++
			},
			deleted: true,
		})

		// Register non-expired entry
		ep.register(&metric{
			expired: false,
			delete: func() {
				deleteCount++
			},
			deleted: true,
		})

		// Register another expired entry
		ep.register(&metric{
			expired: true,
			delete: func() {
				deleteCount++
			},
			deleted: true,
		})

		assert.Equal(t, 3, ep.entries.Len(), "entries not registered")
		ep.check()
		assert.Equal(t, 2, deleteCount, "only expired entries should be deleted")
		assert.Equal(t, 1, ep.entries.Len(), "only non-expired entry should remain")
	})

	t.Run("nil value entry", func(t *testing.T) {
		ep := newExpiryProc(1)

		// Manually add a nil entry to test the nil check
		ep.Lock()
		ep.entries.PushBack(nil)
		ep.Unlock()

		assert.Equal(t, 1, ep.entries.Len(), "nil entry not added")
		ep.check()
		assert.Equal(t, 0, ep.entries.Len(), "nil entry should be removed")
	})
}

func TestExpiryProc_run(t *testing.T) {
	t.Run("run with zero interval returns immediately", func(t *testing.T) {
		ep := newExpiryProc(0)
		ctx := context.Background()

		// This should return immediately without blocking
		done := make(chan bool)
		go func() {
			ep.run(ctx)
			done <- true
		}()

		select {
		case <-done:
			// Success - run returned immediately
		case <-time.After(100 * time.Millisecond):
			t.Fatal("run() should return immediately when interval is 0")
		}
	})

	t.Run("run with context cancellation", func(t *testing.T) {
		ep := newExpiryProc(100 * time.Millisecond)
		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan bool)
		go func() {
			ep.run(ctx)
			done <- true
		}()

		// Give it a moment to start
		time.Sleep(10 * time.Millisecond)

		// Cancel the context
		cancel()

		// Should exit quickly after cancellation
		select {
		case <-done:
			// Success - run exited after context cancellation
		case <-time.After(200 * time.Millisecond):
			t.Fatal("run() should exit when context is cancelled")
		}
	})

	t.Run("run performs periodic checks", func(t *testing.T) {
		ep := newExpiryProc(50 * time.Millisecond)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		deleteCount := 0
		mu := sync.Mutex{}

		// Register an expired metric
		ep.register(&metric{
			expired: true,
			delete: func() {
				mu.Lock()
				deleteCount++
				mu.Unlock()
			},
			deleted: true,
		})

		// Start the run loop
		go ep.run(ctx)

		// Wait for at least one check cycle (interval + 1 second as per run() implementation)
		time.Sleep(1200 * time.Millisecond)

		// Cancel to stop the run loop
		cancel()

		// The metric should have been deleted
		mu.Lock()
		assert.Greater(t, deleteCount, 0, "check() should have been called at least once")
		mu.Unlock()
	})
}

func TestExpiryProc_concurrent_access(t *testing.T) {
	t.Run("concurrent register and check", func(t *testing.T) {
		ep := newExpiryProc(10 * time.Millisecond)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start the run loop
		go ep.run(ctx)

		var wg sync.WaitGroup

		// Concurrently register entries
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ep.register(&metric{
					expired: true,
					deleted: true,
				})
			}()
		}

		wg.Wait()

		// Give time for checks to process (interval + 1 second as per run() implementation)
		time.Sleep(1100 * time.Millisecond)

		// All should be processed and removed
		ep.Lock()
		finalLen := ep.entries.Len()
		ep.Unlock()

		assert.Equal(t, 0, finalLen, "all entries should have been processed")
	})
}
