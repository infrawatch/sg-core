package main

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/infrawatch/sg-core/pkg/data"
	jsoniter "github.com/json-iterator/go"
	"gopkg.in/go-playground/assert.v1"
)

var testMsgsInvalid map[string]string = map[string]string{
	"Null":                    ``,
	"Non-list":                `{}`,
	"Empty":                   `[{}]`,
	"Missing Dstypes":         `[{"values": [2121], "dsnames":["samples"], "host":"localhost","plugin":"metric","type":"type0"}]`,
	"Missing Dsnames":         `[{"values": [2121], "dstypes": ["derive"], "host":"localhost","plugin":"metric","type":"type0"}]`,
	"Missing Values":          `[{"values": [2121], "dstypes": ["derive"], "host":"localhost","plugin":"metric","type":"type0"}]`,
	"Missing Host":            `[{"values": [2121], "dstypes": ["derive"], "dsnames":["samples"],"plugin":"metric","type":"type0"}]`,
	"Missing Plugin":          `[{"values": [2121], "dstypes": ["derive"], "dsnames":["samples"], "host":"localhost","type":"type0"}]`,
	"Missing Type":            `[{"values": [2121], "dstypes": ["derive"], "dsnames":["samples"], "host":"localhost","plugin":"metric"}]`,
	"Inconsistent Dimensions": `[{"values": [2121], "dstypes": ["derive","counter"], "dsnames":["samples"], "host":"localhost","plugin":"metric","type":"type0"}]`,
}

var testMsgsValid map[string]string = map[string]string{
	"Without Instance Types":    `[{"values": [2121], "dstypes": ["derive"], "dsnames":["samples"], "host":"localhost","plugin":"metric","type":"type0"}]`,
	"With Instance Types":       `[{"values": [2121], "dstypes": ["derive"], "dsnames":["samples"], "host":"localhost","plugin_instance":"plugin0","type_instance":"type0", "plugin":"metric","type":"type0"}]`,
	"Multi-dimensional Metrics": `[{"values": [2121], "dstypes": ["derive"], "dsnames":["samples"], "host":"localhost","plugin":"metric","type":"type0"},{"values": [2121, 1010], "dstypes": ["derive","counter"], "dsnames":["samples","samples"], "host":"localhost","plugin":"metric","type":"type0"}]`,
}

var (
	json      = jsoniter.ConfigCompatibleWithStandardLibrary
	metricsUT []data.Metric
)

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

//Use this to update messages in metric-tests-expected.json if behavior should change

// func TestPrintMsgs(t *testing.T) {
// 	metricHandler := New().(*collectdMetricsHandler)
// 	for test, data := range testMsgsValid {
// 		t.Run(test, func(t *testing.T) {
// 			metricHandler.Handle([]byte(data), false, MetricReceive, EventReceive)
// 			blob, _ := json.MarshalIndent(metricsUT, "", "  ")
// 			fmt.Printf("%s\n", string(blob))
// 		})
// 	}
// }

func TestMessageParsing(t *testing.T) {
	expectedData, err := ioutil.ReadFile("messages/metrics-tests-expected.json")
	if err != nil {
		t.Error(err)
	}

	validResults := map[string][]data.Metric{}
	err = json.Unmarshal(expectedData, &validResults)
	if err != nil {
		t.Error(err)
	}

	metricHandler := New().(*collectdMetricsHandler)
	t.Run("Invalid Messages", func(t *testing.T) {
		for _, blob := range testMsgsInvalid {
			metricHandler.totalDecodeErrors = 0
			metricHandler.Handle([]byte(blob), false, MetricReceive, EventReceive)
			assert.Equal(t, uint64(1), metricHandler.totalDecodeErrors)
		}
	})

	metricHandler.totalDecodeErrors = 0
	t.Run("Valid Messages", func(t *testing.T) {
		for test, blob := range testMsgsValid {
			err := metricHandler.Handle([]byte(blob), false, MetricReceive, EventReceive)
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, uint64(0), metricHandler.totalDecodeErrors)
			for index, res := range validResults[test] {
				assert.Equal(t, res, metricsUT[index])
			}
		}
	})
}

// func BenchmarkParsing(b *testing.B) {
// 	// GOMAXPROCS = 8
// 	// On thinkpad T480s, performs at ~ 195k m/s
// 	metricHandler := New().(*collectdMetricsHandler)
// 	for i := 0; i < b.N; i++ {
// 		metricHandler.Handle([]byte(testMsgsValid["Multi-dimensional Metrics"]))
// 	}
// }
