package application

import (
	"context"
	"time"

	"github.com/infrawatch/sg-core/pkg/data"
)

//package application defines the interface for interacting with application plugins

//Application describes application plugin interfaces
type Application interface {
	Config([]byte) error
	RecieveMetric(string, float64, data.MetricType, time.Duration, float64, []string, []string)
	Run(context.Context, chan bool)
}
