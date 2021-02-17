package main

import (
	"bytes"
	"context"
	"fmt"

	"github.com/infrawatch/apputils/connector"
	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/application"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/pkg/errors"

	"github.com/infrawatch/sg-core/plugins/application/loki/pkg/lib"
)

//Print plugin suites for logging both internal buses to a file.
type Loki struct {
	config *lib.LokiConfig
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
		switch hName {
		case "rsyslog-logs":
			log, err := lib.ParseRsyslog(msg, l.logger)
			if err == nil {
				l.logChannel <- *log
			} else {
				l.logger.Metadata(logging.Metadata{"plugin": "loki", "log": msg})
				l.logger.Info("failed to parse the data in event bus - disregarding")

			}
		default:
			l.logger.Metadata(logging.Metadata{"plugin": "loki", "log": msg})
			l.logger.Info("received unknown data in event bus - disregarding")
		}
	default:
		l.logger.Metadata(logging.Metadata{"plugin": "loki", "event": msg})
		l.logger.Info("received event data in event bus - disregarding")
	}
}

//Run run scrape endpoint
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
	l.config = &lib.LokiConfig {
		Connection:  "",
		BatchSize:   20,
		MaxWaitTime: 100,
	}
	err := config.ParseConfig(bytes.NewReader(c), l.config)
	if err != nil {
		return err
	}

	fmt.Println(l.config.Connection)

	l.client, err = lib.NewLokiClient(l.config, l.logger)
	if err != nil {
		return errors.Wrap(err, "failed to connect to Loki host")
	}
	return nil
}
