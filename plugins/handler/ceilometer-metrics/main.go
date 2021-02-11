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
	metricTimeout = 100 //TODO - further research on best interval to use here
)

var (
	ceilTypeToMetricType = map[string]data.MetricType{
		"cumulative": data.COUNTER,
		"delta":      data.UNTYPED,
		"gauge":      data.GAUGE,
	}
)

type ceilometerMetricHandler struct {
	ceilo *ceilometer.Ceilometer
}

func (c *ceilometerMetricHandler) Run(ctx context.Context, mpf bus.MetricPublishFunc, epf bus.EventPublishFunc) {

}

func (c *ceilometerMetricHandler) Handle(blob []byte, reportErrs bool, mpf bus.MetricPublishFunc, epf bus.EventPublishFunc) error {
	msg, err := c.ceilo.ParseInputJSON(blob)
	if err != nil {
		return err
	}

	err = validateMessage(msg)
	if err != nil {
		//TODO: write event to event bus
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
		labelKeys, labelVals := genLabels(&m, msg.Publisher, cNameShards)
		validateMetric(&m, cNameShards)
		mpf(
			genName(&m, cNameShards),
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
		return errors.New("message missing field 'publisher'")
	}

	if len(msg.Payload) == 0 {
		return errors.New("message contains no payload")
	}
	return nil
}

func validateMetric(m *ceilometer.Metric, cNameShards []string) error {
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

func genName(m *ceilometer.Metric, cNameShards []string) string {
	nameParts := []string{"ceilometer"}
	nameParts = append(nameParts, cNameShards...)
	return strings.Join(nameParts, "_")
}

func genLabels(m *ceilometer.Metric, publisher string, cNameShards []string) ([]string, []string) {
	labelKeys := make([]string, 8) //TODO: set to persistant var
	labelVals := make([]string, 8)
	plugin := cNameShards[0]
	pluginVal := m.ResourceID
	if len(cNameShards) > 2 {
		pluginVal = cNameShards[2]
	}
	labelKeys[0] = plugin
	labelVals[0] = pluginVal

	//TODO: should we instead do plugin: <name>, plugin_id: <id> ?

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

//New ceilometer metric handler constructor
func New() handler.Handler {
	return &ceilometerMetricHandler{
		ceilo: ceilometer.New(),
	}
}
