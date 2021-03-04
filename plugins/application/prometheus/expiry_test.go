package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type metric struct {
	delete func()
}

func (m *metric) Expired(i time.Duration) bool {
	return true
}

func (m *metric) Delete() bool {
	m.delete()
	return true
}

func TestExpiry(t *testing.T) {
	ep := newExpiryProc(1)

	t.Run("single entry", func(t *testing.T) {
		deleted := false
		ep.register(&metric{
			delete: func() {
				deleted = true
			},
		})
		assert.Equal(t, 1, ep.entries.Len(), "entry not registered")
		ep.check()
		assert.Equal(t, true, deleted, "expiry.delete() not called")
		assert.Equal(t, 0, ep.entries.Len(), "entry not removed after expiration")
	})
}
