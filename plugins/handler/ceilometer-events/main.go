package main

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/handler"
)

type ceilometerEventsHandler struct {
	totalEventsReceived uint64
}

var (
	rexForNestedQuote  = regexp.MustCompile(`\\\"`)
	rexForOsloMessage  = regexp.MustCompile(`\\*"oslo.message\\*"\s*:\s*\\*"({.*})\\*"`)
	rexForPayload      = regexp.MustCompile(`\\+"payload\\+"\s*:\s*\[(.*)\]`)
	rexForCleanPayload = regexp.MustCompile(`\"payload\"\s*:\s*\[(.*)\]`)
	rexForEventType    = regexp.MustCompile(`\\+"event_type\\+"\s*:\s*\\*"`)
)

func verify(jsondata []byte) bool {
	match := rexForOsloMessage.FindSubmatchIndex(jsondata)
	if match == nil {
		return false
	}
	match = rexForEventType.FindSubmatchIndex(jsondata)
	if match == nil {
		return false
	}
	match = rexForPayload.FindSubmatchIndex(jsondata)
	if match == nil {
		return false
	}
	return true
}

func sanitize(jsondata []byte) string {
	sanitized := string(jsondata)
	// parse only relevant data
	sub := rexForOsloMessage.FindStringSubmatch(sanitized)
	sanitized = rexForNestedQuote.ReplaceAllString(sub[1], `"`)
	// avoid getting payload data wrapped in array
	item := rexForCleanPayload.FindStringSubmatch(sanitized)
	if len(item) == 2 {
		sanitized = rexForCleanPayload.ReplaceAllLiteralString(sanitized, fmt.Sprintf(`"payload":%s`, item[1]))
	}
	return sanitized
}

//Handle implements the data.EventsHandler interface
func (c *ceilometerEventsHandler) Handle(msg []byte, reportErrors bool, mpf bus.MetricPublishFunc, epf bus.EventPublishFunc) error {
	var err error
	c.totalEventsReceived++

	if verify(msg) {
		epf(
			c.Identify(),
			data.EVENT,
			sanitize(msg),
		)
	} else {
		err = errors.New("received message does not have expected format")
		if reportErrors {
			epf(
				c.Identify(),
				data.EVENT,
				fmt.Sprintf(`"error": "%s", "msg": "%s"`, err.Error(), string(msg)),
			)
		}
	}

	return err
}

//Run send internal metrics to bus
func (c *ceilometerEventsHandler) Run(ctx context.Context, mpf bus.MetricPublishFunc, epf bus.EventPublishFunc) {
	for {
		select {
		case <-ctx.Done():
			goto done
		case <-time.After(time.Second * 10):
			mpf(
				"sg_total_ceilometer_events_received",
				0,
				data.COUNTER,
				0,
				float64(c.totalEventsReceived),
				[]string{"source"},
				[]string{"SG"},
			)
		}
	}
done:
}

func (c *ceilometerEventsHandler) Identify() string {
	return "ceilometer-events"
}

//New create new collectdEventsHandler object
func New() handler.Handler {
	return &ceilometerEventsHandler{}
}
