package handler

import (
	"context"

	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/data"
)

// package handler contains the interface description for handler plugins

//MetricHandler mangle messages to place on metric bus
type MetricHandler interface {
	Run(context.Context, bus.PublishFunc)
	Handle([]byte, bus.PublishFunc)
}

//EventHandler mangle messages to place on event bus
type EventHandler interface {
	Handle([]byte) (data.Event, error)
}
