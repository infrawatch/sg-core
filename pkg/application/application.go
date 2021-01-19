package application

import (
	"context"

	"github.com/infrawatch/sg-core/pkg/data"
)

//package application defines the interface for interacting with application plugins

//Application describes application plugin interfaces
type Application interface {
	Config([]byte) error
	Run(context.Context, chan data.Event, chan []data.Metric, chan bool)
}
