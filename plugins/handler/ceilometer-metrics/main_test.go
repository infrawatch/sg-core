package main

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/infrawatch/sg-core/pkg/data"
	jsoniter "github.com/json-iterator/go"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

const (
	metricPrintFormat = "{\n\tname: %s,\n\ttime: %.2f,\n\ttype: %s,\n\tinterval: %d,\n\tvalue: %.2f,\n\tlabelVals: %v,\n\tlabelKeys: %v\n}\n"
)

//CeilometerMetricTemplate holds correct parsings for comparing against parsed results
type CeilometerMetricTestTemplate struct {
	TestInput        jsoniter.RawMessage `json:"testInput"`
	ValidatedResults []*struct {
		Publisher      string            `json:"publisher"`
		Plugin         string            `json:"plugin"`
		PluginInstance string            `json:"plugin_instance"`
		Type           string            `json:"type"`
		TypeInstance   string            `json:"type_instance"`
		Name           string            `json:"name"`
		Key            string            `json:"key"`
		ItemKey        string            `json:"item_Key"`
		Description    string            `json:"description"`
		MetricName     string            `json:"metric_name"`
		Labels         map[string]string `json:"labels"`
		Values         []float64         `json:"values"`
		ISNew          bool
		Interval       float64
	} `json:"validatedResults"`
}

func ceilometerMetricTestTemplateFromJSON(jsonData jsoniter.RawMessage) (*CeilometerMetricTestTemplate, error) {
	var testData CeilometerMetricTestTemplate
	err := json.Unmarshal(jsonData, &testData)
	if err != nil {
		return nil, fmt.Errorf("error parsing json: %s", err)
	}

	for _, r := range testData.ValidatedResults {
		r.Interval = 5.0
		r.ISNew = true
	}
	return &testData, nil
}

func EventReceive(handler string, eType data.EventType, msg string) {
	fmt.Println(handler)
}

func MetricReceive(name string, mTime float64, mType data.MetricType, interval time.Duration, value float64, labelKeys []string, labelVals []string) {
	fmt.Println(name)
	fmt.Printf(metricPrintFormat,
		name,
		mTime,
		mType.String(),
		interval/time.Second,
		value,
		labelKeys,
		labelVals,
	)
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

	// for index, standard := range testCases.ValidatedResults {
	// 	m := mUTs[index]
	// 	assert.Equal(t, standard.Publisher, m.Publisher)
	// 	assert.Equal(t, standard.Plugin, m.Plugin)
	// 	assert.Equal(t, standard.PluginInstance, m.PluginInstance)
	// 	assert.Equal(t, standard.Type, m.Type)
	// 	assert.Equal(t, standard.TypeInstance, m.TypeInstance)
	// 	assert.Equal(t, standard.Values, m.GetValues())
	// 	assert.Equal(t, standard.Interval, m.GetInterval())
	// 	assert.Equal(t, standard.ItemKey, m.GetItemKey())
	// 	assert.Equal(t, standard.Key, m.GetKey())
	// 	assert.Equal(t, standard.Labels, m.GetLabels())
	// 	assert.Equal(t, standard.Name, m.GetName())
	// 	assert.Equal(t, standard.ISNew, m.ISNew())
	// 	assert.Equal(t, standard.Description, m.GetMetricDesc(0))
	// 	assert.Equal(t, standard.MetricName, m.GetMetricName(0))
	// }
}
