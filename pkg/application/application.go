package application

import (
	"context"
	"time"

	"github.com/infrawatch/sg-core/pkg/data"
)

// package application defines the interfaces for interacting with application plugins

//  Application describes application plugin interfaces.
//  Configuration bytes are passed into the Config() function as a sequence of bytes in yaml format. It is recommended to use the config.ParseConfig() method to parse the input. This is a convenience method that uses the validations library to validate the input and provide specific feedback.
//  The main process must be implemented in the Run() method and respect the context.Done() signal. If the plugin wishes to send an exit signal to sg-core, it must send a true value to the boolean channel. This should be done in the case of plugin failure.
type Application interface {
	Config([]byte) error
	Run(context.Context, chan bool)
}

// MetricReceiver Receives metrics from the internal metrics bus
type MetricReceiver interface {
	Application
	//  The ReceiveMetric function will be called every time a Metric is Received on the internal metrics bus. Each part of the metric is passed in as an argument to the function in the following order: name, epoch time, metric type, interval, value, label keys, label values.
	// The last two arguments are guaranteed to be the same size and map index to index. Implementors of this function should run as quickly as possible as metrics can be very high volume. It is recommended to cache metrics in a data.Metric{} object to be utilized by the application plugin later.
	ReceiveMetric(
		string, // name
		float64, // epoch time
		data.MetricType, // type
		time.Duration, // interval
		float64, // value
		[]string, // labelKeys
		[]string, // labelValues
	)
}

// EventReceiver Receive events from the internal event bus
type EventReceiver interface {
	Application
	// ReceiveEvent is called whenever an event is broadcast on the event bus.
	ReceiveEvent(data.Event)
}
