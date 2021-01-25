package main

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"

	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/handler"
)

type collectdEventsHandler struct{}

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

//Handle implements the data.collectdEventsHandler interface
func (c *collectdEventsHandler) Handle(msg []byte, reportErrors bool) (data.Event, error) {
	var err error
	event := data.Event{Handler: c.Identify()}

	if verify(msg) {
		event.Type = data.EVENT
		event.Message = sanitize(bytes.Trim(msg, "\t []"))
	} else {
		message := fmt.Sprintf("received message does not have expected format")
		err = errors.New(message)
		if reportErrors {
			event.Type = data.ERROR
			event.Message = fmt.Sprintf(`"error": "%s", "msg": "%s"`, message, msg)
		}
	}

	return event, err
}

func (c *collectdEventsHandler) Identify() string {
	return "collectd-events"
}

//New create new collectdEventsHandler object
func New() handler.EventHandler {
	return &collectdEventsHandler{}
}
