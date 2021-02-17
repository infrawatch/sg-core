package main

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"
	"bytes"

	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/handler"
)

type rsyslogLogsHandler struct {
	totalLogsReceived uint64
}

var (
	// Regular expressions for identifying rsyslog logs
	rexForTimestamp = regexp.MustCompile(`"@timestamp":`)
	rexForSeverity = regexp.MustCompile(`"severity":`)
	rexForFacility = regexp.MustCompile(`"facility":`)
)

func verify(data []byte) bool {
	timestamp := rexForTimestamp.FindIndex(data)
	severity := rexForSeverity.FindIndex(data)
	facility := rexForFacility.FindIndex(data)

	return timestamp != nil && severity != nil && facility != nil
}

//Handle implements the data.EventsHandler interface
func (r *rsyslogLogsHandler) Handle(msg []byte, reportErrors bool, mpf bus.MetricPublishFunc, epf bus.EventPublishFunc) error {
	var err error
	r.totalLogsReceived++

	if verify(msg) {
		msgLength := bytes.IndexByte(msg, 0)
		epf(
			r.Identify(),
			data.LOG,
			string(msg[:msgLength]),
		)
	} else {
		err = errors.New("received message does not have expected format")
		if reportErrors {
			epf(
				r.Identify(),
				data.ERROR,
				fmt.Sprintf(`"error": "%s", "msg": "%s"`, err.Error(), string(msg)),
			)
		}
	}

	return err
}

//Run send internal metrics to bus
func (r *rsyslogLogsHandler) Run(ctx context.Context, mpf bus.MetricPublishFunc, epf bus.EventPublishFunc) {
	for {
		select {
		case <-ctx.Done():
			goto done
		case <-time.After(time.Second):
			mpf(
				"sg_total_rsyslog_logs_received",
				0,
				data.COUNTER,
				0,
				float64(r.totalLogsReceived),
				[]string{"source"},
				[]string{"SG"},
			)
		}
	}
done:
}

func (r *rsyslogLogsHandler) Identify() string {
	return "rsyslog-logs"
}

//New create new rsyslogLogsHandler object
func New() handler.Handler {
	return &rsyslogLogsHandler{}
}
