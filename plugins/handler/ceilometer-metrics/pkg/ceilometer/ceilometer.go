package ceilometer

import (
	"fmt"
	"regexp"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/vmihailenco/msgpack/v5"
)

var (
	rexForPayload     = regexp.MustCompile(`\"payload\"\s*:\s*\[(.*)\]`)
	rexForNestedQuote = regexp.MustCompile(`\\\"`)
	json              = jsoniter.ConfigCompatibleWithStandardLibrary
)

// Metedata represents metadataof a metric from ceilometer
type Metadata struct {
	Host         string            `json:"host" msgpack:"host"`
	Name         string            `json:"name" msgpack:"name"`
	DisplayName  string            `json:"display_name" msgpack:"display_name"`
	InstanceHost string            `json:"instance_host" msgpack:"instance_host"`
	UserMetadata map[string]string `json:"user_metadata" msgpack:"user_metadata"`
}

// Metric represents a single metric from ceilometer for unmarshalling
type Metric struct {
	Source           string   `json:"source" msgpack:"source"`
	CounterName      string   `json:"counter_name" msgpack:"counter_name"`
	CounterType      string   `json:"counter_type" msgpack:"counter_type"`
	CounterUnit      string   `json:"counter_unit" msgpack:"counter_unit"`
	CounterVolume    float64  `json:"counter_volume" msgpack:"counter_volume"`
	UserID           string   `json:"user_id" msgpack:"user_id"`
	UserName         string   `json:"user_name" msgpack:"user_name"`
	ProjectID        string   `json:"project_id" msgpack:"project_id"`
	ProjectName      string   `json:"project_name" msgpack:"project_name"`
	ResourceID       string   `json:"resource_id" msgpack:"resource_id"`
	Timestamp        string   `json:"timestamp" msgpack:"timestamp"`
	ResourceMetadata Metadata `json:"resource_metadata" msgpack:"resource_metadata"`
}

// Message struct represents an incoming ceilometer metrics message
type Message struct {
	Publisher string   `json:"publisher_id" msgpack:"publisher_id"`
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

// ParseInputMsgPack parse blob into list of metrics
func (c *Ceilometer) ParseInputMsgPack(blob []byte) (*Message, error) {
	msg := &Message{}
	metric := Metric{}
	err := msgpack.Unmarshal(blob, &metric)
	if err != nil {
		return nil, err
	}
	err = msgpack.Unmarshal(blob, msg)
	if err != nil {
		return nil, err
	}
	msg.Payload = append(msg.Payload, metric)
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
