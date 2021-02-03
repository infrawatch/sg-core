package main

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/handler"
)

type ceilometerEventsHandler struct{}

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
func (c *ceilometerEventsHandler) Handle(msg []byte, reportErrors bool) (*data.Event, error) {
	var err error
	event := &data.Event{Handler: c.Identify()}

	if verify(msg) {
		event.Type = data.EVENT
		event.Message = sanitize(msg)
	} else {
		message := fmt.Sprintf("received message does not have expected format")
		err = errors.New(message)
		if reportErrors {
			event.Type = data.ERROR
			event.Message = fmt.Sprintf(`"error": "%s", "msg": "%s"`, message, string(msg))
		} else {
			event = nil
		}
	}

	return event, err
}

func (c *ceilometerEventsHandler) Identify() string {
	return "ceilometer-events"
}

//New create new collectdEventsHandler object
func New() handler.EventHandler {
	return &ceilometerEventsHandler{}
}
