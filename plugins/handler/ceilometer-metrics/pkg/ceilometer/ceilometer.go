package ceilometer

import (
	"fmt"
	"regexp"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

var (
	rexForPayload     = regexp.MustCompile(`\"payload\"\s*:\s*\[(.*)\]`)
	rexForNestedQuote = regexp.MustCompile(`\\\"`)
	json              = jsoniter.ConfigCompatibleWithStandardLibrary
)

// Metedata represents metadataof a metric from ceilometer
type Metadata struct {
	Host string
}

// Metric represents a single metric from ceilometer for unmarshalling
type Metric struct {
	Source           string
	CounterName      string  `json:"counter_name"`
	CounterType      string  `json:"counter_type"`
	CounterUnit      string  `json:"counter_unit"`
	CounterVolume    float64 `json:"counter_volume"`
	UserID           string  `json:"user_id"`
	ProjectID        string  `json:"project_id"`
	ResourceID       string  `json:"resource_id"`
	Timestamp        string
	ResourceMetadata Metadata `json:"resource_metadata"`
}

// Message struct represents an incoming ceilometer metrics message
type Message struct {
	Publisher string   `json:"publisher_id"`
	Payload   []Metric `json:"payload"`
}

// OsloSchema initial OsloSchema
type OsloSchema struct {
	Request struct {
		OsloMessage string `json:"oslo.message"`
	}
}

// Ceilometer instance for parsing and handling ceilometer metric messages
type Ceilometer struct {
	schema OsloSchema
}

// New Ceilometer constructor
func New() *Ceilometer {
	return &Ceilometer{
		schema: OsloSchema{},
	}
}

// ParseInputJSON parse blob into list of metrics
func (c *Ceilometer) ParseInputJSON(blob []byte) (*Message, error) {
	msg := &Message{}
	err := json.Unmarshal(blob, &c.schema)
	if err != nil {
		return nil, err
	}
	sanitized := c.sanitize()
	err = json.Unmarshal([]byte(sanitized), &msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// sanitize remove extraneous characters
func (c *Ceilometer) sanitize() string {
	sanitized := rexForNestedQuote.ReplaceAllString(c.schema.Request.OsloMessage, `"`)

	item := rexForPayload.FindStringSubmatch(sanitized)
	if len(item) == 2 {
		sanitized = rexForPayload.ReplaceAllString(sanitized, fmt.Sprintf(`"payload": [%s]`, strings.Join(item[1:], ",")))
	}

	return sanitized
}
