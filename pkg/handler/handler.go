package handler

import (
	"context"

	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/data"
)

// package handler contains the interface description for handler plugins

//MetricHandler mangle messages to place on metric bus
type MetricHandler interface {
	//Run should only be used to send metrics apart from those being parsed from the transport. For example, this process could send metrics tracking the number of arrived messages and send them to the bus on a time delayed interval
	Run(context.Context, bus.MetricPublishFunc)

	//Handle parse incoming messages from the transport and write resulting metrics to the metric bus
	Handle([]byte, bus.MetricPublishFunc)
}

//EventHandler mangle messages to place on event bus
type EventHandler interface {
	Handle([]byte) (data.Event, error)
}
