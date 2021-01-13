package application

import (
	"context"
	"sync"

	"github.com/infrawatch/sg-core-refactor/pkg/data"
)

//package application defines the interface for interacting with application plugins

//Application describes application plugin interfaces
type Application interface {
	Config([]byte) error
	Run(context.Context, *sync.WaitGroup, chan data.Event, chan []data.Metric, chan bool)
}
