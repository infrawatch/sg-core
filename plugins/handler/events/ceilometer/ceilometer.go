package ceilometer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/plugins/handler/events/pkg/lib"
	jsoniter "github.com/json-iterator/go"
)

var (
	// Regular expression for sanitizing received data
	rexForNestedQuote = regexp.MustCompile(`\\\"`)
	// json parser
	json = jsoniter.ConfigCompatibleWithStandardLibrary
	// severity converter. A DNE returns data.INFO
	ceilometerAlertSeverity = map[string]data.EventSeverity{
		"audit":    data.INFO,
		"info":     data.INFO,
		"sample":   data.INFO,
		"warn":     data.WARNING,
		"warning":  data.WARNING,
		"critical": data.CRITICAL,
		"error":    data.CRITICAL,
		"AUDIT":    data.INFO,
		"INFO":     data.INFO,
		"SAMPLE":   data.INFO,
		"WARN":     data.WARNING,
		"WARNING":  data.WARNING,
		"CRITICAL": data.CRITICAL,
		"ERROR":    data.CRITICAL,
	}
)

const (
	genericSuffix = "_generic"
	source        = "ceilometer"
)

type rawMessage struct {
	Request struct {
		OsloVersion string `json:"oslo.version"`
		OsloMessage string `json:"oslo.message"`
	}
}

func (rm *rawMessage) sanitizeMessage() {
	// sets oslomessage to cleaned state
	rm.Request.OsloMessage = rexForNestedQuote.ReplaceAllLiteralString(rm.Request.OsloMessage, `"`)
}

type osloPayload struct {
	MessageID string `json:"message_id"`
	EventType string `json:"event_type"`
	Generated string `json:"generated"`
	Traits    []interface{}
}

func (op *osloPayload) traitsFormatted() (map[string]interface{}, error) {
	// transforms traits key into map[string]interface{}
	newTraits := make(map[string]interface{})
	for _, value := range op.Traits {
		if typedValue, ok := value.([]interface{}); ok {
			if len(typedValue) != 3 {
				return nil, fmt.Errorf("parsed invalid trait in event: '%v'", value)
			}
			if traitType, ok := typedValue[1].(float64); ok {
				switch traitType {
				case 2:
					newTraits[typedValue[0].(string)] = typedValue[2].(float64)
				default:
					newTraits[typedValue[0].(string)] = typedValue[2].(string)
				}
			} else {
				return nil, fmt.Errorf("parsed invalid trait in event: '%v'", value)
			}
		} else {
			return nil, fmt.Errorf("parsed invalid trait in event: '%v'", value)
		}
	}
	return newTraits, nil
}

type osloMessage struct {
	EventType   string `json:"event_type"`
	PublisherID string `json:"publisher_id"`
	Timestamp   string
	Priority    string
	Payload     []osloPayload
}

func (om *osloMessage) fromBytes(blob []byte) error {
	return json.Unmarshal(blob, om)
}

// Ceilometer holds parsed ceilometer event data and provides methods for retrieving that data
// in a standardizes format
type Ceilometer struct {
	osloMessage osloMessage
}

// Parse parse ceilometer message data
func (c *Ceilometer) Parse(blob []byte) error {
	rm := rawMessage{}
	err := json.Unmarshal(blob, &rm)
	if err != nil {
		return err
	}
	rm.sanitizeMessage()

	c.osloMessage = osloMessage{}
	return c.osloMessage.fromBytes([]byte(rm.Request.OsloMessage))
}

func (c *Ceilometer) name(index int) string {
	// use event_type from payload or fallback to message's event_type if N/A
	if c.osloMessage.Payload[index].EventType != "" {
		return buildName(c.osloMessage.Payload[index].EventType)
	}

	if c.osloMessage.EventType != "" {
		return buildName(c.osloMessage.EventType)
	}
	return buildName(fmt.Sprintf("%s%s", source, genericSuffix))
}

// PublishEvents iterate through events in payload calling publish function on each iteration
func (c *Ceilometer) PublishEvents(epf bus.EventPublishFunc) error {
	for idx, event := range c.osloMessage.Payload {
		ts, err := event.traitsFormatted()
		if err != nil {
			return err
		}
		epf(data.Event{

			Index:     c.name(idx),
			Time:      c.getTimeAsEpoch(event),
			Type:      data.EVENT,
			Publisher: c.osloMessage.PublisherID,
			Severity:  ceilometerAlertSeverity[c.osloMessage.Priority],
			Labels:    ts,
			Annotations: map[string]interface{}{
				"source_type":  source,
				"processed_by": "sg",
			},
		})
	}
	return nil
}

func (c *Ceilometer) getTimeAsEpoch(payload osloPayload) float64 {
	// order of precedence: payload timestamp, message timestamp, zero

	if payload.Generated != "" {
		return float64(lib.EpochFromFormat(payload.Generated))
	}

	if c.osloMessage.Timestamp != "" {
		return float64(lib.EpochFromFormat(c.osloMessage.Timestamp))
	}

	return 0.0
}

func buildName(eventType string) string {
	var output string
	etParts := strings.Split(eventType, ".")
	if len(etParts) > 1 {
		output = strings.Join(etParts[:len(etParts)-1], "_")
	}
	output = strings.ReplaceAll(output, "-", "_")

	// ensure index name is prefixed with source name
	if !strings.HasPrefix(output, fmt.Sprintf("%s_", source)) {
		output = fmt.Sprintf("%s_%s", source, output)
	}
	return output
}
