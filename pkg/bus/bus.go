package bus

import (
	"sync"
	"time"

	"github.com/infrawatch/sg-core/pkg/data"
)

//EventReceiveFunc callback type for receiving events from the event bus
// arguments: handler name, event type, message
type EventReceiveFunc func(string, data.EventType, string)

//EventPublishFunc function to for publishing to the event bus
// arguments: handler name, event type, message
type EventPublishFunc func(string, data.EventType, string)

//EventBus bus for data.Event type
type EventBus struct {
	subscribers []EventReceiveFunc
	rw          sync.RWMutex
}

//Subscribe subscribe to bus
func (eb *EventBus) Subscribe(rf EventReceiveFunc) {
	eb.rw.Lock()
	defer eb.rw.Unlock()
	eb.subscribers = append(eb.subscribers, rf)
}

//Publish publish to bus
func (eb *EventBus) Publish(hName string, eType data.EventType, msg string) {
	eb.rw.RLock()

	for _, rf := range eb.subscribers {
		go func(rf EventReceiveFunc) {
			rf(hName, eType, msg)
		}(rf)
	}
	eb.rw.RUnlock() //defer is actually very slow
}

// MetricReceiveFunc callback type for receiving metrics
// arguments are name, timestamp, metric type, interval, value, labels
type MetricReceiveFunc func(string, float64, data.MetricType, time.Duration, float64, []string, []string)

//MetricPublishFunc function type for publishing to the metric bus
type MetricPublishFunc func(string, float64, data.MetricType, time.Duration, float64, []string, []string)

//MetricBus bus for data.Metric type
type MetricBus struct {
	sync.RWMutex
	subscribers []MetricReceiveFunc
}

//Subscribe subscribe to bus
func (mb *MetricBus) Subscribe(rf MetricReceiveFunc) {
	mb.Lock()
	defer mb.Unlock()
	mb.subscribers = append(mb.subscribers, rf)
}

//Publish publish to bus
func (mb *MetricBus) Publish(name string, time float64, mType data.MetricType, interval time.Duration, value float64, labelKeys []string, labelVals []string) {
	mb.RLock()
	for _, rf := range mb.subscribers {
		go func(rf MetricReceiveFunc) {
			rf(name, time, mType, interval, value, labelKeys, labelVals)
		}(rf)
	}
	mb.RUnlock()
}
