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

// Event internal event type
type Event struct {
	Handler string
	Message string
}

// Metric internal metric type
type Metric struct {
	Name     string
	Labels   map[string]string
	Time     time.Time
	Type     MetricType
	Interval time.Duration
	Value    float64
}
