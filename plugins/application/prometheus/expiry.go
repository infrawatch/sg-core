package main

import (
	"container/list"
	"context"
	"sync"
	"time"
)

// expire metrics after time of staleness
// for infinite time, set interval to 0

type expiry interface {
	Expired(time.Duration) bool
	Delete() bool
}

type expiryProc struct {
	sync.Mutex
	entries  *list.List
	interval time.Duration
}

func newExpiryProc(interval time.Duration) *expiryProc {
	return &expiryProc{
		entries:  list.New(),
		interval: interval,
	}
}

func (ep *expiryProc) register(e expiry) {
	ep.Lock()
	defer ep.Unlock()
	ep.entries.PushBack(e)
}

func (ep *expiryProc) check() {
	ep.Lock()
	defer ep.Unlock()

	e := ep.entries.Front()
	for e != nil {
		// NOTE(vkmc) Shouldn't be required with the lock in place
		if e.Value == nil {
			next := e.Next()
			ep.entries.Remove(e)
			e = next
			continue
		}

		expirable := e.Value.(expiry)
		if expirable.Expired(ep.interval) {
			if expirable.Delete() {
				next := e.Next()
				ep.entries.Remove(e)
				e = next
				continue
			}
		}
		e = e.Next()
	}
}

func (ep *expiryProc) run(ctx context.Context) {
	if ep.interval == 0 {
		return
	}

	for {
		select {
		case <-ctx.Done():
			goto done
		case <-time.After(ep.interval + time.Second):
			ep.check()
		}
	}
done:
}
