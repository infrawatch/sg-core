package data

import (
	"time"
)

// package data defines the data descriptions for objects used in the internal buses

//----------------------------------- events ----------------------------------

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
	// LOG event contains log record
	LOG
	// RESULT event contains data about result of check execution
	// perfomed by any supported client side agent (collectd-sensubility, sg-agent)
	RESULT
	// TASK contains request of performing some task, for example scheduler app asking transport to send message
	TASK
)

func (et EventType) String() string {
	return []string{"error", "event", "log", "result", "task"}[et]
}

// EventSeverity indicates severity of an event
type EventSeverity int

const (
	//UNKNOWN ... default
	UNKNOWN EventSeverity = iota
	//INFO ...
	INFO
	//WARNING ...
	WARNING
	//CRITICAL ...
	CRITICAL
)

func (es EventSeverity) String() string {
	return []string{"unknown", "info", "warning", "critical"}[es]
}

// Event convenience type that contains all elements of an event on the bus. This type is good to use for caching and testing
type Event struct {
	Index       string
	Time        float64
	Type        EventType
	Publisher   string
	Severity    EventSeverity
	Labels      map[string]interface{}
	Annotations map[string]interface{}
}

//---------------------------------- metrics ----------------------------------

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

// Metric internal metric type
type Metric struct {
	Name      string
	Time      float64
	Type      MetricType
	Interval  time.Duration
	Value     float64
	LabelKeys []string
	LabelVals []string
}
