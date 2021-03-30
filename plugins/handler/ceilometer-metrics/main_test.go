package main

import (
	"fmt"
	"io/ioutil"
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

// type Metric struct {
// 	Source           string
// 	CounterName      string  `json:"counter_name"`
// 	CounterType      string  `json:"counter_type"`
// 	CounterUnit      string  `json:"counter_unit"`
// 	CounterVolume    float64 `json:"counter_volume"`
// 	UserID           string  `json:"user_id"`
// 	ProjectID        string  `json:"project_id"`
// 	ResourceID       string  `json:"resource_id"`
// 	Timestamp        string
// 	ResourceMetadata metadata `json:"resource_metadata"`
// }
func TestCeilometerIncoming(t *testing.T) {
	plugin := New()

	testData, err := ioutil.ReadFile("messages/metric-tests.json")
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

		labelKeys, _ := genLabels(metric, "node-0", []string{"volume", "size"})

		fmt.Println(labelKeys)
		// should have 8 labels
		assert.Equal(t, len(labelKeys), 8)

	})

}
