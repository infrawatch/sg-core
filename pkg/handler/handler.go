package handler

import "github.com/infrawatch/sg-core/pkg/data"

// package handler contains the interface description for handler plugins

//MetricHandler mangle messages to place on metric bus
type MetricHandler interface {
	Handle([]byte) []data.Metric
}

//EventHandler mangle messages to place on event bus
type EventHandler interface {
	Handle([]byte) (data.Event, error)
}
