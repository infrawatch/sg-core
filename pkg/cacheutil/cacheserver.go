package cacheutil

import (
	"container/list"
	"context"
	"time"
)

// Expiry use to free memory after expire condition
type Expiry interface {
	Expired() bool
	Delete()
}

// CacheServer for now used only to expire Expiry types
type CacheServer struct {
	entries  *list.List
	Interval time.Duration
}

// NewCacheServer CacheServer factory that sets expiry interval in seconds
func NewCacheServer() *CacheServer {
	return &CacheServer{
		entries:  list.New(),
		Interval: 5,
	}
}

// Register new expiry object
func (cs *CacheServer) Register(e Expiry) {
	cs.entries.PushBack(e)
}

// Run run cache server
func (cs *CacheServer) Run(ctx context.Context) error {
	// expiry loop

	var err error
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			goto done
		default:
			e := cs.entries.Front()
			for {
				if e == nil {
					break
				}

				if e.Value.(Expiry).Expired() {
					e.Value.(Expiry).Delete()
					n := e.Next()
					cs.entries.Remove(e)
					e = n
					continue
				}
				e = e.Next()
			}
			time.Sleep(time.Second * cs.Interval)
		}
	}
done:
	return err
}
