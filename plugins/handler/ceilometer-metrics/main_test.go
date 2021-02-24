package main

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/infrawatch/sg-core/pkg/data"
	jsoniter "github.com/json-iterator/go"
	"gopkg.in/go-playground/assert.v1"
)

var (
	json      = jsoniter.ConfigCompatibleWithStandardLibrary
	metricsUT []data.Metric
)

//CeilometerMetricTemplate holds correct parsings for comparing against parsed results
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

func EventReceive(data.Event)

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

func TestCeilometerIncoming(t *testing.T) {
	plugin := New()

	testData, err := ioutil.ReadFile("messages/metric-tests.json")
	if err != nil {
		t.Errorf("failed loading test data: %s", err.Error())
	}

	tests := make(map[string]jsoniter.RawMessage)
	err = json.Unmarshal([]byte(testData), &tests)
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
