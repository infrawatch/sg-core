package main

import (
	"bytes"
	"context"
	"fmt"
	"time"
	"encoding/json"

	"github.com/infrawatch/apputils/connector"
	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/application"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/pkg/errors"

	"github.com/infrawatch/sg-core/plugins/application/loki/pkg/lib"
)

type LokiConfig struct {
	Connection  string `validate:"required"`
	BatchSize   int64
	MaxWaitTime time.Duration
}

//Loki plugin for forwarding logs to loki
type Loki struct {
	config *LokiConfig
	client *connector.LokiConnector
	logger *logging.Logger
	logChannel chan interface{}
}

//New constructor
func New(logger *logging.Logger) application.Application {
	return &Loki {
		logger:     logger,
		logChannel: make(chan interface{}, 100),
	}
}

// ReceiveEvent ...
func (l *Loki) ReceiveEvent(hName string, eType data.EventType, msg string) {
	switch eType {
	case data.LOG:
		log, err := lib.CreateLokiLog(msg)
		a, err := json.Marshal(log)
		fmt.Println(string(a))
		if err == nil {
			l.logChannel <- log
		} else {
			l.logger.Metadata(logging.Metadata{"plugin": "loki", "log": msg})
			l.logger.Info("failed to parse the data in event bus - disregarding")

		}
	default:
		l.logger.Metadata(logging.Metadata{"plugin": "loki", "event": msg})
		l.logger.Info("received event data (instead of log data) in event bus - disregarding")
	}
}

//Run run loki application plugin
func (l *Loki) Run(ctx context.Context, done chan bool) {
	l.logger.Metadata(logging.Metadata{"plugin": "loki", "url": l.config.Connection})
	l.logger.Info("storing logs to loki.")
	l.client.Start(nil, l.logChannel)

	<-ctx.Done()
	l.client.Disconnect()

	l.logger.Metadata(logging.Metadata{"plugin": "loki"})
	l.logger.Info("exited")
}

//Config implements application.Application
func (l *Loki) Config(c []byte) error {
	l.config = &LokiConfig {
		Connection:  "",
		BatchSize:   20,
		MaxWaitTime: 100,
	}
	err := config.ParseConfig(bytes.NewReader(c), l.config)
	if err != nil {
		return err
	}
	fmt.Println(l.config.MaxWaitTime)

	l.client, err = connector.CreateLokiConnector(l.logger,
	                                              l.config.Connection,
	                                              l.config.MaxWaitTime,
	                                              l.config.BatchSize)
	if err != nil {
		return errors.Wrap(err, "failed to connect to Loki host")
	}
	return nil
}
