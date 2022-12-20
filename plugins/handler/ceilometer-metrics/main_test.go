package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/plugins/handler/ceilometer-metrics/pkg/ceilometer"
	jsoniter "github.com/json-iterator/go"
	"gopkg.in/go-playground/assert.v1"
)

var (
	json      = jsoniter.ConfigCompatibleWithStandardLibrary
	metricsUT []data.Metric
)

var expectedMsgpackMetric = data.Metric{
	Name:      "ceilometer_test_name_0_0_82",
	Time:      0,
	Type:      data.UNTYPED,
	Interval:  100 * time.Second,
	Value:     0,
	LabelKeys: []string{"test_name_0_0_82", "publisher", "type", "counter", "project", "unit", "resource"},
	LabelVals: []string{"test_resource_id", "localhost.localdomain", "test_name_0_0_82", "test_name_0_0_82", "test_project_id_0", "test_unit", "test_resource_id"},
}

// CeilometerMetricTemplate holds correct parsings for comparing against parsed results
type CeilometerMetricTestTemplate struct {
	TestInput        jsoniter.RawMessage `json:"testInput"`
	ValidatedResults []data.Metric       `json:"validatedResults"`
}

func ceilometerMetricTestTemplateFromJSON(jsonData jsoniter.RawMessage) (*CeilometerMetricTestTemplate, error) {
	var testData CeilometerMetricTestTemplate
	err := json.Unmarshal(jsonData, &testData)
	if err != nil {
		return nil, fmt.Errorf("error parsing json: %s", err)
	}
	return &testData, nil
}

func EventReceive(data.Event) {

}

func MetricReceive(name string, mTime float64, mType data.MetricType, interval time.Duration, value float64, labelKeys []string, labelVals []string) {
	metricsUT = append(metricsUT, data.Metric{
		Name:      name,
		Time:      mTime,
		Type:      mType,
		Interval:  interval,
		Value:     value,
		LabelKeys: labelKeys,
		LabelVals: labelVals,
	})
}

func TestCeilometerIncomingJSON(t *testing.T) {
	plugin := New()
	err := plugin.Config([]byte{})
	if err != nil {
		t.Errorf("failed configuring ceilometer handler plugin: %s", err.Error())
	}

	testData, err := os.ReadFile("messages/metric-tests.json")
	if err != nil {
		t.Errorf("failed loading test data: %s", err.Error())
	}

	tests := make(map[string]jsoniter.RawMessage)
	err = json.Unmarshal(testData, &tests)
	if err != nil {
		t.Errorf("failed to unmarshal test data: %s", err.Error())
	}

	inputData, ok := tests["CeilometerMetrics"]
	if !ok {
		t.Error("'CeilometerMetrics' field not found in test data")
	}
	testCases, err := ceilometerMetricTestTemplateFromJSON(inputData)
	if err != nil {
		t.Error(err)
	}

	err = plugin.Handle(testCases.TestInput, false, MetricReceive, EventReceive)
	if err != nil {
		t.Error(err)
	}

	for index, expMetric := range testCases.ValidatedResults {
		expMetric.Interval = time.Second * metricTimeout
		assert.Equal(t, expMetric, metricsUT[index])
	}
}
func TestCeilometerIncomingMsgpack(t *testing.T) {
	plugin := New()
	err := plugin.Config([]byte("source: tcp"))
	if err != nil {
		t.Errorf("failed configuring ceilometer handler plugin: %s", err.Error())
	}

	testData, err := os.ReadFile("messages/msgpack-test.msgpack")
	if err != nil {
		t.Errorf("failed loading test data: %s", err.Error())
	}

	metricsUT = []data.Metric{}
	err = plugin.Handle(testData, false, MetricReceive, EventReceive)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, expectedMsgpackMetric, metricsUT[0])
}

func TestGenLabelsSizes(t *testing.T) {
	t.Run("un-exhaustive labels", func(t *testing.T) {
		// ensure slices are correct length when the parsed message does not contain all of the noncritical label fields

		metric := ceilometer.Metric{
			Source:        "openstack",
			CounterName:   "volume.size",
			CounterType:   "gauge",
			CounterUnit:   "GB",
			CounterVolume: 2,
			UserID:        "user_id",
			ProjectID:     "db3fce7b7aeb4109bb2794f9337e68fa",
			ResourceID:    "ed8102c3-923a-4f5a-9a24-d59afc174755",
			Timestamp:     "2021-03-30T15:20:19.891893",
		}

		labelKeys, labelVals := genLabels(metric, "node-0", []string{"volume", "size"})

		// must always be same size since they represent a map
		assert.Equal(t, len(labelKeys), len(labelVals))

		// cannot have empty labelKey entries
		for _, key := range labelKeys {
			if key == "" {
				t.Error("zero-value key in label keys")
			}
		}

		// should have 7 labels
		assert.Equal(t, len(labelKeys), 7)
	})

	t.Run("exhaustive labels", func(t *testing.T) {
		metric := ceilometer.Metric{
			Source:        "openstack",
			CounterName:   "volume.size",
			CounterType:   "gauge",
			CounterUnit:   "GB",
			CounterVolume: 2,
			UserID:        "user_id",
			ProjectID:     "db3fce7b7aeb4109bb2794f9337e68fa",
			ResourceID:    "ed8102c3-923a-4f5a-9a24-d59afc174755",
			Timestamp:     "2021-03-30T15:20:19.891893",
			ResourceMetadata: ceilometer.Metadata{
				Host: "host1",
			},
		}

		labelKeys, labelVals := genLabels(metric, "node-0", []string{"volume", "size"})

		// must always be same size since they represent a map
		assert.Equal(t, len(labelKeys), len(labelVals))

		fmt.Println(labelKeys)
		// should have 8 labels
		assert.Equal(t, len(labelKeys), 8)

	})

}
