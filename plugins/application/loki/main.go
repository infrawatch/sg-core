package main

import (
	"bytes"
	"context"
	"time"

	"github.com/infrawatch/apputils/connector/loki"
	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/application"
	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/pkg/errors"

	"github.com/infrawatch/sg-core/plugins/application/loki/pkg/lib"
)

// LokiConfig halds plugin configuration
type LokiConfig struct {
	Connection  string        `validate:"required"`
	BatchSize   int64         `yaml:"batchSize"`
	MaxWaitTime time.Duration `yaml:"maxWaitTime"`
}

// Loki plugin for forwarding logs to loki
type Loki struct {
	config     *LokiConfig
	client     *loki.LokiConnector
	logger     *logging.Logger
	logChannel chan interface{}
}

// New constructor
func New(logger *logging.Logger, sendMetric bus.MetricPublishFunc, sendEvent bus.EventPublishFunc) application.Application {
	return &Loki{
		logger:     logger,
		logChannel: make(chan interface{}, 100),
	}
}

// ReceiveEvent ...
func (l *Loki) ReceiveEvent(log data.Event) {
	switch log.Type {
	case data.LOG:
		lokiLog, err := lib.CreateLokiLog(log)
		if err != nil {
			l.logger.Metadata(logging.Metadata{"plugin": "loki", "log": log, "error": err})
			l.logger.Error("failed to parse the data in event bus - disregarding")
			return
		}
		l.logChannel <- lokiLog
	default:
		l.logger.Metadata(logging.Metadata{"plugin": "loki", "event": log})
		l.logger.Error("received event data (instead of log data) in event bus - disregarding")
	}
}

// Run run loki application plugin
func (l *Loki) Run(ctx context.Context, done chan bool) {
	l.logger.Metadata(logging.Metadata{"plugin": "loki", "url": l.config.Connection})
	l.logger.Info("storing logs to Loki.")
	l.client.Start(nil, l.logChannel)

	<-ctx.Done()
	l.client.Disconnect()

	l.logger.Metadata(logging.Metadata{"plugin": "loki"})
	l.logger.Info("exited")
}

// Config implements application.Application
func (l *Loki) Config(c []byte) error {
	l.config = &LokiConfig{
		Connection:  "",
		BatchSize:   20,
		MaxWaitTime: 100,
	}
	err := config.ParseConfig(bytes.NewReader(c), l.config)
	if err != nil {
		return err
	}

	l.client, err = loki.CreateLokiConnector(l.logger,
		l.config.Connection,
		l.config.MaxWaitTime,
		l.config.BatchSize,
		"fake")
	if err != nil {
		return errors.Wrap(err, "failed to connect to Loki host")
	}
	return nil
}
