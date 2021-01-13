package main

import (
	"container/list"
	"context"
	"time"
)

//expire metrics after time of staleness

type expiry interface {
	Expired() bool
	Delete()
}

type expiryProc struct {
	entries  *list.List
	interval time.Duration
}

func newExpiryProc() *expiryProc {
	return &expiryProc{
		entries:  list.New(),
		interval: 10000,
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

		if e.Value.(expiry).Expired() {
			e.Value.(expiry).Delete()
			n := e.Next()
			ep.entries.Remove(e)
			e = n
			continue
		}
		e = e.Next()
	}
}

func (ep *expiryProc) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			goto done
		case <-time.After(time.Millisecond * ep.interval):
			ep.check()
		}
	}
done:
}
