package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/handler"
	"github.com/infrawatch/sg-core/plugins/handler/ceilometer-metrics/pkg/ceilometer"
)

const (
	metricTimeout = 100 // TODO - further research on best interval to use here
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
	msg, err := c.ceilo.ParseInputJSON(blob)
	if err != nil {
		return err
	}

	err = validateMessage(msg)
	if err != nil {
		c.totalDecodeErrors++
		if reportErrs {
			epf(data.Event{ //THIS IS EXTREMELY SLOW
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

		mType := ceilTypeToMetricType[m.CounterType] //zero value is UNTYPED

		cNameShards := strings.Split(m.CounterName, ".")
		labelKeys, labelVals := genLabels(m, msg.Publisher, cNameShards)
		err = validateMetric(m, cNameShards)
		if err != nil {
			c.totalDecodeErrors++
			if reportErrs {
				epf(data.Event{
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
		c.totalMetricsDecoded++
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

func validateMetric(m ceilometer.Metric, cNameShards []string) error {
	if len(cNameShards) < 1 {
		return errors.New("missing 'counter_name' in metric payload")
	}

	if m.ProjectID == "" {
		return errors.New("metric missing 'project_id'")
	}

	if m.ResourceID == "" {
		return errors.New("metric missing 'resource_id'")
	}

	if m.CounterName == "" {
		return errors.New("metric missing 'counter_name'")
	}

	if m.CounterUnit == "" {
		return errors.New("metric missing 'counter_unit'")
	}

	if m.ResourceMetadata.Host == "" {
		return errors.New("metric missing 'resource_metadata.host'")
	}

	return nil
}

func genName(cNameShards []string) string {
	nameParts := []string{"ceilometer"}
	nameParts = append(nameParts, cNameShards...)
	return strings.Join(nameParts, "_")
}

func genLabels(m ceilometer.Metric, publisher string, cNameShards []string) ([]string, []string) {
	labelKeys := make([]string, 8) // TODO: set to persistent var
	labelVals := make([]string, 8)
	plugin := cNameShards[0]
	pluginVal := m.ResourceID
	if len(cNameShards) > 2 {
		pluginVal = cNameShards[2]
	}
	labelKeys[0] = plugin
	labelVals[0] = pluginVal

	// TODO: should we instead do plugin: <name>, plugin_id: <id> ?

	labelKeys[1] = "publisher"
	labelVals[1] = publisher

	labelKeys[2] = "counter"
	labelVals[2] = m.CounterName

	var ctype string
	if len(cNameShards) > 1 {
		ctype = cNameShards[1]
	} else {
		ctype = cNameShards[0]
	}
	labelKeys[3] = "type"
	labelVals[3] = ctype

	labelKeys[4] = "project"
	labelVals[4] = m.ProjectID

	labelKeys[5] = "unit"
	labelVals[5] = m.CounterUnit

	labelKeys[6] = "resource"
	labelVals[6] = m.ResourceID

	labelKeys[7] = "host"
	labelVals[7] = m.ResourceMetadata.Host

	return labelKeys, labelVals
}

func (c *ceilometerMetricHandler) Identify() string {
	return "ceilometer-metrics"
}

func (c *ceilometerMetricHandler) Config(blob []byte) error {
	return nil
}

// New ceilometer metric handler constructor
func New() handler.Handler {
	return &ceilometerMetricHandler{
		ceilo: ceilometer.New(),
	}
}
