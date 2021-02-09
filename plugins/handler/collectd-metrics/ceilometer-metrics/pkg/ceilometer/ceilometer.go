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
)

//Metric struct represents a single instance of metric data formated and sent by Ceilometer
type Metric struct {
	Publisher string                 `json:"publisher_id"`
	Payload   map[string]interface{} `json:"payload"`
	// analogy to collectd metric
	Plugin         string
	PluginInstance string
	Type           string
	TypeInstance   string
	Values         []float64
	new            bool
	wholeID        string
}

//OsloSchema initial OsloSchema
type OsloSchema struct {
	Request struct {
		OsloMessage string `json:"oslo.message"`
	}
}

//Ceilometer instance for parsing and handling ceilometer metric messages
type Ceilometer struct {
	schema OsloSchema
}

//New Ceilometer constructor
func New() *Ceilometer {
	return &Ceilometer{
		schema: OsloSchema{},
	}
}

//ParseInputJSON parse blob into list of metrics
func (c *Ceilometer) ParseInputJSON(blob []byte) ([]Metric, error) {
	ceil := []Metric{}
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	err := json.Unmarshal(blob, &c.schema)
	if err != nil {
		return nil, err
	}
	sanitzed := c.sanitize()

	return ceil, nil
}

//sanitize remove extraneous characters
func (c *Ceilometer) sanitize() string {
	sanitized := rexForNestedQuote.ReplaceAllString(c.schema.Request.OsloMessage, `"`)

	item := rexForPayload.FindStringSubmatch(sanitized)
	if len(item) == 2 {
		sanitized = rexForPayload.ReplaceAllString(sanitized, fmt.Sprintf(`"payload": [%s]`, strings.Join(item[1:], ",")))
	}

	return sanitized
}
