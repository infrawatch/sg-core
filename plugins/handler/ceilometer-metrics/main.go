package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/handler"
	"github.com/infrawatch/sg-core/plugins/handler/ceilometer-metrics/pkg/ceilometer"
)

const (
	metricTimeout = 100 //  TODO - further research on best interval to use here
)

var (
	ceilTypeToMetricType = map[string]data.MetricType{
		"cumulative": data.COUNTER,
		"delta":      data.UNTYPED,
		"gauge":      data.GAUGE,
	}
)

type ceilometerMetricHandler struct {
	ceilo                 *ceilometer.Ceilometer
	totalMetricsDecoded   uint64
	totalDecodeErrors     uint64
	totalMessagesReceived uint64
	config                ceilometerConfig
}

// The tcp and udp ceilometer publishers send the data in a message pack format.
// The messaging ceilometer publisher sends the data in a JSON format.
// That's the reason why we need to know the source.
type ceilometerConfig struct {
	Source string `yaml:"source"`
}

func (c *ceilometerMetricHandler) Run(ctx context.Context, mpf bus.MetricPublishFunc, epf bus.EventPublishFunc) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
			mpf(
				"sg_total_ceilometer_metric_decode_count",
				0,
				data.COUNTER,
				0,
				float64(c.totalMetricsDecoded),
				[]string{"source"},
				[]string{"SG"},
			)
			mpf(
				"sg_total_ceilometer_metric_decode_error_count",
				0,
				data.COUNTER,
				0,
				float64(c.totalDecodeErrors),
				[]string{"source"},
				[]string{"SG"},
			)
			mpf(
				"sg_total_ceilometer_msg_received_count",
				0,
				data.COUNTER,
				0,
				float64(c.totalMessagesReceived),
				[]string{"source"},
				[]string{"SG"},
			)
		}
	}

}

func (c *ceilometerMetricHandler) Handle(blob []byte, reportErrs bool, mpf bus.MetricPublishFunc, epf bus.EventPublishFunc) error {
	c.totalMessagesReceived++
	var msg *ceilometer.Message
	var err error
	switch c.config.Source {
	case "tcp":
		fallthrough
	case "udp":
		msg, err = c.ceilo.ParseInputMsgPack(blob)
	case "unix":
		fallthrough
	default:
		msg, err = c.ceilo.ParseInputJSON(blob)
	}
	if err != nil {
		return err
	}

	err = validateMessage(msg)
	if err != nil {
		c.totalDecodeErrors++
		if reportErrs {
			epf(data.Event{ // THIS IS EXTREMELY SLOW
				Index:    c.Identify(),
				Type:     data.ERROR,
				Severity: data.CRITICAL,
				Time:     0.0,
				Labels: map[string]interface{}{
					"error":   err.Error(),
					"message": "failed to parse metric - disregarding",
				},
				Annotations: map[string]interface{}{
					"description": "internal smartgateway ceilometer-metrics handler error",
				},
			})
		}
		return err
	}

	var gTime time.Time
	var t float64
	for _, m := range msg.Payload {
		gTime, _ = time.Parse(time.RFC3339, m.Timestamp)
		t = float64(gTime.Unix())
		if t < 0.0 {
			t = 0.0
		}

		mType := ceilTypeToMetricType[m.CounterType] // zero value is UNTYPED
		if m.CounterName == "" {
			c.totalDecodeErrors++
			if reportErrs {
				epf(data.Event{
					Index:    c.Identify(),
					Type:     data.ERROR,
					Severity: data.CRITICAL,
					Time:     0.0,
					Labels: map[string]interface{}{
						"error":   "missing 'counter_name' in metric payload",
						"message": "failed to parse metric - disregarding",
					},
					Annotations: map[string]interface{}{
						"description": "internal smartgateway ceilometer-metrics handler error",
					},
				})
			}
			return errors.New("missing 'counter_name' in metric payload")
		}

		c.totalMetricsDecoded++
		cNameShards := strings.Split(m.CounterName, ".")
		labelKeys, labelVals := genLabels(m, msg.Publisher, cNameShards)
		mpf(
			genName(cNameShards),
			t,
			mType,
			time.Second*metricTimeout,
			m.CounterVolume,
			labelKeys,
			labelVals,
		)
	}

	return nil
}

func validateMessage(msg *ceilometer.Message) error {
	if msg.Publisher == "" {
		return errors.New("message missing field 'publisher_id'")
	}

	if len(msg.Payload) == 0 {
		return errors.New("message contains no payload")
	}
	return nil
}

func genName(cNameShards []string) string {
	nameParts := []string{"ceilometer"}
	nameParts = append(nameParts, cNameShards...)
	return strings.Join(nameParts, "_")
}

func genLabels(m ceilometer.Metric, publisher string, cNameShards []string) ([]string, []string) {
	//  TODO: set to persistent var
	labelKeys := make([]string, 12+len(m.ResourceMetadata.UserMetadata))
	labelVals := make([]string, 12+len(m.ResourceMetadata.UserMetadata))
	plugin := cNameShards[0]
	pluginVal := m.ResourceID
	if len(cNameShards) > 2 {
		pluginVal = cNameShards[2]
	}
	labelKeys[0] = plugin
	labelVals[0] = pluginVal

	//  TODO: should we instead do plugin: <name>, plugin_id: <id> ?

	labelKeys[1] = "publisher"
	labelVals[1] = publisher

	var ctype string
	if len(cNameShards) > 1 {
		ctype = cNameShards[1]
	} else {
		ctype = cNameShards[0]
	}

	labelKeys[2] = "type"
	labelVals[2] = ctype

	// non - critical
	index := 3
	if m.CounterName != "" {
		labelKeys[index] = "counter"
		labelVals[index] = m.CounterName
		index++
	}

	if m.ProjectID != "" {
		labelKeys[index] = "project"
		labelVals[index] = m.ProjectID
		index++
	}

	if m.ProjectName != "" {
		labelKeys[index] = "project_name"
		labelVals[index] = m.ProjectName
		index++
	}

	if m.UserID != "" {
		labelKeys[index] = "user"
		labelVals[index] = m.UserID
		index++
	}

	if m.UserName != "" {
		labelKeys[index] = "user_name"
		labelVals[index] = m.UserName
		index++
	}

	if m.CounterUnit != "" {
		labelKeys[index] = "unit"
		labelVals[index] = m.CounterUnit
		index++
	}

	if m.ResourceID != "" {
		labelKeys[index] = "resource"
		labelVals[index] = m.ResourceID
		index++
	}

	if m.ResourceMetadata.Host != "" {
		labelKeys[index] = "vm_instance"
		labelVals[index] = m.ResourceMetadata.Host
		index++
	}

	if m.ResourceMetadata.DisplayName != "" {
		labelKeys[index] = "resource_name"
		labelVals[index] = m.ResourceMetadata.DisplayName
		// index++
	}

	if m.ResourceMetadata.Name != "" {
		labelKeys[index] = "resource_name"
		if labelVals[index] != "" {
			// Use the ":" delimiter when DisplayName is not None
			labelVals[index] = labelVals[index] + ":" + m.ResourceMetadata.Name
		} else {
			labelVals[index] = m.ResourceMetadata.Name
		}
	}
	if labelVals[index] != "" {
		index++
	}
	if len(m.ResourceMetadata.UserMetadata) != 0 {
		for key, value := range m.ResourceMetadata.UserMetadata {
			labelKeys[index] = key
			labelVals[index] = value
			index++
		}
	}

	return labelKeys[:index], labelVals[:index]
}

func (c *ceilometerMetricHandler) Identify() string {
	return "ceilometer-metrics"
}

func (c *ceilometerMetricHandler) Config(blob []byte) error {
	c.config = ceilometerConfig{
		Source: "unix",
	}
	err := config.ParseConfig(bytes.NewReader(blob), &c.config)
	if err != nil {
		return err
	}

	c.config.Source = strings.ToLower(c.config.Source)

	if c.config.Source != "unix" && c.config.Source != "tcp" && c.config.Source != "udp" {
		return fmt.Errorf("incorrect source, should be either \"unix\", \"tcp\" or \"udp\", received: %s",
			c.config.Source)
	}
	return nil
}

// New ceilometer metric handler constructor
func New() handler.Handler {
	return &ceilometerMetricHandler{
		ceilo: ceilometer.New(),
	}
}
