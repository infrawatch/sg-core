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

// example message
/*
{
  "request": {
    "oslo.version": "2.0",
    "oslo.message": "{\"message_id\": \"36ce5025-85b4-4dcf-bce1-757cb96acf90\", \"publisher_id\": \"telemetry.publisher.controller-0.redhat.local\", \"event_type\": \"metering\", \"priority\": \"SAMPLE\", \"payload\": [{\"source\": \"openstack\", \"counter_name\": \"network.outgoing.packets\", \"counter_type\": \"cumulative\", \"counter_unit\": \"packet\", \"counter_volume\": 14, \"user_id\": \"581d4d733fad4baebd0edecb5dc6b889\", \"project_id\": \"40390761eaf7414c8125917efc21024c\", \"resource_id\": \"instance-00000001-13d97b93-63c6-4518-a885-05b95199ae9d-tapd042b21c-a9\", \"timestamp\": \"2021-02-08T20:45:28.286586\", \"resource_metadata\": {\"display_name\": \"BLUE\", \"name\": \"tapd042b21c-a9\", \"instance_id\": \"13d97b93-63c6-4518-a885-05b95199ae9d\", \"instance_type\": \"m1.tiny\", \"host\": \"34a0c5f56e8cc42025727379a8640387bb6c8e37b916b82940354c42\", \"instance_host\": \"compute-1.redhat.local\", \"flavor\": {\"id\": \"beeaf8e5-2bda-4b3c-88ad-cf41ad4df149\", \"name\": \"m1.tiny\", \"vcpus\": 2, \"ram\": 512, \"disk\": 1, \"ephemeral\": 0, \"swap\": 0}, \"status\": \"active\", \"state\": \"running\", \"task_state\": \"\", \"image\": null, \"image_ref\": null, \"image_ref_url\": null, \"architecture\": \"x86_64\", \"os_type\": \"hvm\", \"vcpus\": 2, \"memory_mb\": 512, \"disk_gb\": 1, \"ephemeral_gb\": 0, \"root_gb\": 1, \"mac\": \"fa:16:3e:92:a3:86\", \"fref\": null, \"parameters\": {\"interfaceid\": \"d042b21c-a92d-41cb-bc73-cb906cc151e7\", \"bridge\": \"br-int\"}, \"vnic_name\": \"tapd042b21c-a9\"}, \"message_id\": \"93b28638-6a4e-11eb-8ea0-52540021b407\", \"monotonic_time\": null, \"message_signature\": \"9db3fe4203b30f20073fd437f1e5b148faafe7029670a0cd7149657611233ce4\"}], \"timestamp\": \"2021-02-08 21:23:06.418913\"}"
  },
  "context": {}
}
*/

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
		t = float64(gTime.Unix()) //TODO: test that this equals zero when time.Parse returns an error

		mType := ceilTypeToMetricType[m.CounterType] //zero value is UNTYPED

		cNameShards := strings.Split(m.CounterName, ".")
		labelKeys, labelVals := genLabels(&m, msg.Publisher, cNameShards)
		mpf(
			genName(&m, cNameShards),
			t,
			mType,
			time.Second*10, //TODO: further exploration into what this should be
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

	return nil
}

func genName(m *ceilometer.Metric, cNameShards []string) string {
	nameParts := []string{"ceilometer"}
	nameParts = append(nameParts, cNameShards...)
	return strings.Join(nameParts, "_")
}

func genLabels(m *ceilometer.Metric, publisher string, cNameShards []string) ([]string, []string) {
	labelKeys := make([]string, 7) //TODO: set to persistant var
	labelVals := make([]string, 7)
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
