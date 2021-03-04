package main

import (
	"container/list"
	"context"
	"time"
)

// expire metrics after time of staleness
// for infinite time, set interval to 0

type expiry interface {
	Expired(time.Duration) bool
	Delete() bool
}

type expiryProc struct {
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
	ep.entries.PushBack(e)
}

func (ep *expiryProc) check() {
	e := ep.entries.Front()
	for {
		if e == nil {
			break
		}

		if e.Value.(expiry).Expired(ep.interval) {
			if e.Value.(expiry).Delete() {
				n := e.Next()
				ep.entries.Remove(e)
				e = n
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
