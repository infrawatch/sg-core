package bus

import (
	"sync"
	"time"

	"github.com/infrawatch/sg-core/pkg/data"
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

//RecieveFunc callback type for receiving metrics
// Arguments are name, timestamp, metric type, interval, value, labels
type MetricRecieveFunc func(string, float64, data.MetricType, time.Duration, float64, []string, []string)

//PublishFunc ...
type MetricPublishFunc func(string, float64, data.MetricType, time.Duration, float64, []string, []string)

//MetricBus bus for data.Metric type
type MetricBus struct {
	sync.RWMutex
	subscribers []MetricRecieveFunc
}

//Subscribe subscribe to bus
func (mb *MetricBus) Subscribe(rf MetricRecieveFunc) {
	mb.Lock()
	defer mb.Unlock()
	mb.subscribers = append(mb.subscribers, rf)
}

//Publish publish to bus
func (mb *MetricBus) Publish(name string, time float64, typ data.MetricType, interval time.Duration, value float64, labelKeys []string, labelVals []string) {
	mb.RLock()
	for _, rf := range mb.subscribers {
		go func(rf MetricRecieveFunc) {
			rf(name, time, typ, interval, value, labelKeys, labelVals)
		}(rf)
	}
	mb.RUnlock()
}
