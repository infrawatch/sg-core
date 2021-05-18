package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/handler"
	"github.com/infrawatch/sg-core/plugins/handler/logs/pkg/lib"
)

type logHandler struct {
	totalLogsReceived uint64
	config            lib.LogConfig
}

func (l *logHandler) parse(log []byte) (data.Event, error) {
	parsedLog := data.Event{}
	logFields := make(map[string]interface{})
	err := json.Unmarshal(log, &logFields)
	if err != nil {
		return parsedLog, err
	}

	msg, ok := logFields[l.config.MessageField].(string)
	if !ok {
		return parsedLog, fmt.Errorf("unable to find a log message under field called: %s", l.config.MessageField)
	}

	slSeverity := lib.GetSeverityFromLog(logFields, l.config)

	hostname, ok := logFields[l.config.HostnameField].(string)
	if !ok {
		return parsedLog, fmt.Errorf("unable to find the hostname under field called: %s", l.config.HostnameField)
	}

	timestring, ok := logFields[l.config.TimestampField].(string)
	if !ok {
		return parsedLog, fmt.Errorf("unable to find the timestamp under field called: %s", l.config.TimestampField)
	}
	t, err := lib.TimeFromFormat(timestring)
	if err != nil {
		return parsedLog, err
	}

	timestamp := float64(t.Unix())
	year, month, day := t.Date()

	index := fmt.Sprintf("%s-%s.%d.%02d.%02d", l.config.IndexPrefix, strings.ReplaceAll(hostname, "-", "_"), year, month, day)

	// remove message and timestamp from labels (leave the rest)
	delete(logFields, l.config.MessageField)
	delete(logFields, l.config.TimestampField)

	evtSeverity := slSeverity.ToEventSeverity()
	parsedLog = data.Event{
		Index:     index,
		Time:      timestamp,
		Type:      data.LOG,
		Publisher: hostname,
		Severity:  evtSeverity,
		Labels:    logFields,
		Message:   msg,
	}
	return parsedLog, nil
}

// Handle implements the data.EventsHandler interface
func (l *logHandler) Handle(msg []byte, reportErrors bool, mpf bus.MetricPublishFunc, epf bus.EventPublishFunc) error {
	var err error
	l.totalLogsReceived++

	if log, err := l.parse(msg); err == nil {
		epf(log)
	} else if reportErrors {
		epf(data.Event{
			Index:    l.Identify(),
			Type:     data.ERROR,
			Severity: data.CRITICAL,
			Time:     0.0,
			Labels: map[string]interface{}{
				"error":   err.Error(),
				"context": string(msg),
				"message": "failed to parse log - disregarding",
			},
			Annotations: map[string]interface{}{
				"description": "internal smartgateway log handler error",
			},
		})
	}

	return err
}

// Run send internal metrics to bus
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

// New create new logHandler object
func New() handler.Handler {
	return &logHandler{
		totalLogsReceived: 0,
		config: lib.LogConfig{
			CorrectSeverity: false,
			IndexPrefix:     "sglogs",
		},
	}
}

func (l *logHandler) Config(c []byte) error {
	l.config = lib.LogConfig{
		CorrectSeverity: false,
		IndexPrefix:     "sglogs",
	}
	return config.ParseConfig(bytes.NewReader(c), &l.config)
}
