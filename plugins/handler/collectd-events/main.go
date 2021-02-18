package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/handler"
)

type collectdEventsHandler struct {
	totalEventsReceived uint64
}

var (
	// Regular expressions for identifying collectd events
	rexForLabelsField     = regexp.MustCompile(`\\?"labels\\?"\w?:\w?\{`)
	rexForAnnotationField = regexp.MustCompile(`\\?"annotations\\?"\w?:\w?\{`)
	// Regular expression for sanitizing received data
	rexForInvalidVesStr  = regexp.MustCompile(`":"[^",\\]+"[^",\\]+"`)
	rexForRemainedNested = regexp.MustCompile(`":"[^",]+\\\\\"[^",]+"`)
	rexForNestedQuote    = regexp.MustCompile(`\\\"`)
	rexForVes            = regexp.MustCompile(`"ves":"{(.*)}"`)
)

func verify(jsondata []byte) bool {
	labels := rexForLabelsField.FindIndex(jsondata)
	annots := rexForAnnotationField.FindIndex(jsondata)
	return labels != nil && annots != nil
}

func sanitize(jsondata []byte) string {
	output := string(jsondata)
	// sanitize "ves" field which can come in nested string in more than one level
	sub := rexForVes.FindStringSubmatch(output)
	if len(sub) == 2 {
		substr := sub[1]
		for {
			cleaned := rexForNestedQuote.ReplaceAllString(substr, `"`)
			if rexForInvalidVesStr.FindString(cleaned) == "" {
				substr = cleaned
			}
			if rexForRemainedNested.FindString(cleaned) == "" {
				break
			}
		}
		output = rexForVes.ReplaceAllLiteralString(output, fmt.Sprintf(`"ves":{%s}`, substr))
	}
	return output
}

//Handle implements the data.EventsHandler interface
func (c *collectdEventsHandler) Handle(msg []byte, reportErrors bool, mpf bus.MetricPublishFunc, epf bus.EventPublishFunc) error {
	var err error
	c.totalEventsReceived++

	if verify(msg) {
		epf(
			c.Identify(),
			data.EVENT,
			sanitize(bytes.Trim(msg, "\t []")),
		)
	} else {
		err = errors.New("received message does not have expected format")
		if reportErrors {
			epf(
				c.Identify(),
				data.ERROR,
				fmt.Sprintf(`"error": "%s", "msg": "%s"`, err.Error(), msg),
			)
		}
	}

	return err
}

//Run implements handler.Handler
func (c *collectdEventsHandler) Run(ctx context.Context, mpf bus.MetricPublishFunc, epf bus.EventPublishFunc) {
	for {
		select {
		case <-ctx.Done():
			goto done
		case <-time.After(time.Second):
			mpf(
				"sg_total_collectd_events_received",
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
	return
}

func (c *collectdEventsHandler) Identify() string {
	return "collectd-events"
}

func (c *collectdEventsHandler) Config(blob []byte) error {
	return nil
}

//New create new collectdEventsHandler object
func New() handler.Handler {
	return &collectdEventsHandler{}
}
