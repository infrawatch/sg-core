package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/application"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
)

type configT struct {
	MetricOutput string
	EventsOutput string
}

//Print plugin suites for logging both internal buses to a file.
type Print struct {
	configuration configT
	logger        *logging.Logger
}

//New constructor
func New(logger *logging.Logger) application.Application {
	return &Print{
		configuration: configT{
			MetricOutput: "/dev/stdout",
			EventsOutput: "/dev/stdout",
		},
		logger: logger,
	}
}

//Run run scrape endpoint
func (print *Print) Run(ctx context.Context, wg *sync.WaitGroup, eChan chan data.Event, mChan chan []data.Metric, done chan bool) {
	defer wg.Done()
	wg.Add(1)

	metrF, err := os.OpenFile(print.configuration.MetricOutput, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		print.logger.Metadata(logging.Metadata{"plugin": "print", "error": err})
		print.logger.Error("failed to open metrics data output file")
	} else {
		defer metrF.Close()
	}

	evtsF, errr := os.OpenFile(print.configuration.EventsOutput, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		print.logger.Metadata(logging.Metadata{"plugin": "print", "error": err})
		print.logger.Error("failed to open events data output file")
	} else {
		defer evtsF.Close()
	}

	if err == nil && errr == nil {
		print.logger.Metadata(logging.Metadata{"plugin": "print", "events": print.configuration.EventsOutput, "metrics": print.configuration.MetricOutput})
		print.logger.Info("writing processed data to files.")

		for {
			select {
			case <-ctx.Done():
				goto done
			case event := <-eChan:
				encoded, err := json.MarshalIndent(event, "", "  ")
				if err != nil {
					print.logger.Metadata(logging.Metadata{"plugin": "print", "data": event})
					print.logger.Warn("failed to marshal event data")
				}
				evtsF.WriteString(fmt.Sprintf("Processed event:\n%s\n", string(encoded)))
			case metrics := <-mChan:
				encoded, err := json.MarshalIndent(metrics, "", "  ")
				if err != nil {
					print.logger.Metadata(logging.Metadata{"plugin": "print", "data": metrics})
					print.logger.Warn("failed to marshal metric data")
				}
				metrF.WriteString(fmt.Sprintf("Processed metric:\n%s\n", string(encoded)))
			}
		}
	}
done:
	_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	print.logger.Metadata(logging.Metadata{"plugin": "print"})
	print.logger.Info("exited")
}

//Config implements application.Application
func (print *Print) Config(c []byte) error {
	print.configuration = configT{}
	err := config.ParseConfig(bytes.NewReader(c), &print.configuration)
	if err != nil {
		return err
	}
	//default values
	if print.configuration.EventsOutput == "" {
		print.configuration.EventsOutput = "/dev/stdout"
	}
	if print.configuration.MetricOutput == "" {
		print.configuration.MetricOutput = "/dev/stdout"
	}
	return nil
}
