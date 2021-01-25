package data

import (
	"time"
)

// package data defines the data descriptions for objects used in the internal buses

// MetricType follows standard metric conventions from prometheus
type MetricType int

const (
	//COUNTER ...
	COUNTER MetricType = iota
	//GAUGE ...
	GAUGE
	//UNTYPED ...
	UNTYPED
)

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

// Event internal event type
type Event struct {
	Handler string
	Type    EventType
	Message string
}

// Metric internal metric type
type Metric struct {
	Name      string
	Labels    map[string]string
	LabelKeys []string
	LabelVals []string
	Time      float64
	Type      MetricType
	Interval  time.Duration
	Value     float64
}
