package bus

import (
	"sync"

	"github.com/infrawatch/sg-core-refactor/pkg/data"
)

/* TODO: optimize this
channels are not particularily fast, best to avoid them in the fastpath and use Mutexes instead
benchmark implementation with callback vs channels here
*/

//EventBus bus for data.Event type
type EventBus struct {
	subscribers []chan data.Event
	rw          sync.RWMutex
}

//Subscribe subscribe to bus
func (eb *EventBus) Subscribe(c chan data.Event) {
	eb.rw.Lock()
	defer eb.rw.Unlock()
	eb.subscribers = append(eb.subscribers, c)
}

//Publish publish to bus
func (eb *EventBus) Publish(e data.Event) {
	eb.rw.RLock()
	for _, c := range eb.subscribers {
		go func(c chan data.Event, e data.Event) {
			c <- e
		}(c, e)
	}
	eb.rw.RUnlock() //defer is actually very slow
}

//MetricBus bus for data.Metric type
type MetricBus struct {
	subscribers []chan []data.Metric
	rw          sync.RWMutex
}

//Subscribe subscribe to bus
func (mb *MetricBus) Subscribe(c chan []data.Metric) {
	mb.rw.Lock()
	defer mb.rw.Unlock()
	mb.subscribers = append(mb.subscribers, c)
}

//Publish publish to bus
func (mb *MetricBus) Publish(m []data.Metric) {
	mb.rw.RLock()
	for _, c := range mb.subscribers {
		go func(c chan []data.Metric, m []data.Metric) {
			c <- m
		}(c, m)
	}
	mb.rw.RUnlock() //defer is actually very slow
}
