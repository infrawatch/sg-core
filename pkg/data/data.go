package data

import (
	"time"
)

// package data defines the data descriptions for objects used in the internal buses

// MetricType follows standard metric conventions from prometheus
type MetricType int

const (
	//UNTYPED ...
	UNTYPED MetricType = iota
	//COUNTER only increases in value
	COUNTER
	//GAUGE can increase or decrease in value
	GAUGE
)

func (mt MetricType) String() string {
	return []string{"untyped", "counter", "gauge"}[mt]
}

// EventType marks type of data held in event message
type EventType int

const (
	// ERROR event contains handler failure data and should be handled on application level
	ERROR EventType = iota
	// EVENT contains regular event data
	EVENT
	// RESULT event contains data about result of check execution
	// perfomed by any supported client side agent (collectd-sensubility, sg-agent)
	RESULT
	// LOG event contains log record
	LOG
)

func (et EventType) String() string {
	return []string{"error", "event", "result", "log"}[et]
}

// Event convenience type that contains all elements of an event on the bus. This type is good to use for caching and testing
type Event struct {
	Handler string
	Type    EventType
	Message string
}

// Metric convenience type that contains all elements of a metric on the bus. This type is good to use for caching and testing
type Metric struct {
	Name      string
	Time      float64
	Type      MetricType
	Interval  time.Duration
	Value     float64
	LabelKeys []string
	LabelVals []string
}
