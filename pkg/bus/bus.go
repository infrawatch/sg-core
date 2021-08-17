package bus

import (
	"sync"
	"time"

	"github.com/infrawatch/sg-core/pkg/data"
)

// EventReceiveFunc callback type for receiving events from the event bus
type EventReceiveFunc func(data.Event)

// EventPublishFunc function to for publishing to the event bus
type EventPublishFunc func(data.Event)

// EventBus bus for data.Event type
type EventBus struct {
	subscribers []EventReceiveFunc
	rw          sync.RWMutex
	wg          sync.WaitGroup
}

// Subscribe subscribe to bus
func (eb *EventBus) Subscribe(rf EventReceiveFunc) {
	eb.rw.Lock()
	defer eb.rw.Unlock()
	eb.subscribers = append(eb.subscribers, rf)
}

// Publish publish to bus
func (eb *EventBus) Publish(e data.Event) {
	eb.rw.RLock()

	for _, rf := range eb.subscribers {
		go func(rf EventReceiveFunc) {
			rf(e)
		}(rf)
	}
	eb.rw.RUnlock()
}

// PublishBlocking publish to bus, but block
// until all application plugins process the data
// before publishing more events.
func (eb *EventBus) PublishBlocking(e data.Event) {
	eb.rw.RLock()

	for _, rf := range eb.subscribers {
		eb.wg.Add(1)
		go func(rf EventReceiveFunc, wg *sync.WaitGroup) {
			defer wg.Done()
			rf(e)
		}(rf, &eb.wg)
	}
	eb.wg.Wait()
	eb.rw.RUnlock()
}

// MetricReceiveFunc callback type for receiving metrics
// arguments are name, timestamp, metric type, interval, value, labels
type MetricReceiveFunc func(string, float64, data.MetricType, time.Duration, float64, []string, []string)

// MetricPublishFunc function type for publishing to the metric bus
type MetricPublishFunc func(string, float64, data.MetricType, time.Duration, float64, []string, []string)

// MetricBus bus for data.Metric type
type MetricBus struct {
	sync.RWMutex
	subscribers []MetricReceiveFunc
}

// Subscribe subscribe to bus
func (mb *MetricBus) Subscribe(rf MetricReceiveFunc) {
	mb.Lock()
	defer mb.Unlock()
	mb.subscribers = append(mb.subscribers, rf)
}

// Publish publish to bus
func (mb *MetricBus) Publish(name string, time float64, mType data.MetricType, interval time.Duration, value float64, labelKeys []string, labelVals []string) {
	mb.RLock()
	for _, rf := range mb.subscribers {
		go func(rf MetricReceiveFunc) {
			rf(name, time, mType, interval, value, labelKeys, labelVals)
		}(rf)
	}
	mb.RUnlock()
}
