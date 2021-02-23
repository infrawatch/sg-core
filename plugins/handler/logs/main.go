package main

import (
	"context"
	"fmt"
	"time"
	"bytes"
	"encoding/json"

	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/handler"
)

type logConfig struct {
	MessageField   string `validate:"required"`
	TimestampField string `validate:"required"`
}
type logHandler struct {
	totalLogsReceived uint64
	config            logConfig
}

type logFormat struct {
	Message   string
	Timestamp time.Time
	Tags      map[string]string
}

func (l *logHandler) parse(log []byte) ([]byte, error) {
	data := make(map[string]string)
	err := json.Unmarshal(log, &data)
	if err != nil {
		return nil, err
	}

	msg := data[l.config.MessageField]
	timestamp, err := time.Parse(time.RFC3339, data[l.config.TimestampField])

	if err != nil {
		return nil, err
	}

	delete(data, l.config.MessageField)
	delete(data, l.config.TimestampField)

	parsedLog := logFormat {
		Message:   msg,
		Timestamp: timestamp,
		Tags:      data,
	}

	return json.Marshal(parsedLog)
}

//Handle implements the data.EventsHandler interface
func (l *logHandler) Handle(msg []byte, reportErrors bool, mpf bus.MetricPublishFunc, epf bus.EventPublishFunc) error {
	var err error
	l.totalLogsReceived++

	msgLength := bytes.IndexByte(msg, 0)
	log, err := l.parse(msg[:msgLength])
	if err == nil {
		epf(
			l.Identify(),
			data.LOG,
			string(log),
		)
	} else {
		if reportErrors {
			epf(
				l.Identify(),
				data.ERROR,
				fmt.Sprintf(`"error": "%s", "msg": "%s"`, err.Error(), string(msg)),
			)
		}
	}

	return err
}

//Run send internal metrics to bus
func (l *logHandler) Run(ctx context.Context, mpf bus.MetricPublishFunc, epf bus.EventPublishFunc) {
	for {
		select {
		case <-ctx.Done():
			goto done
		case <-time.After(time.Second):
			mpf(
				"sg_total_logs_received",
				0,
				data.COUNTER,
				0,
				float64(l.totalLogsReceived),
				[]string{"source"},
				[]string{"SG"},
			)
		}
	}
done:
}

func (l *logHandler) Identify() string {
	return "log"
}

//New create new logHandler object
func New() handler.Handler {
	return &logHandler{}
}

func (l *logHandler) Config(c []byte) error {
	l.config = logConfig{}
	return config.ParseConfig(bytes.NewReader(c), &l.config)
}
