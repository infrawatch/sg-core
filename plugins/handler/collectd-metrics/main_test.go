package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/stretchr/testify/assert"
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
	"Without Instance Types":    `[{"values": [2121], "dstypes": ["derive"], "dsnames":["samples"], "host":"localhost", "plugin":"metric", "type":"type0"}]`,
	"With Instance Types":       `[{"values": [2122], "dstypes": ["derive"], "dsnames":["samples"], "host":"localhost", "plugin_instance":"plugin0", "type_instance":"type0", "plugin":"metric", "type":"type666"}]`,
	"Multi-dimensional Metrics": `[{"values": [2112, 1001], "dstypes": ["derive","counter"], "dsnames":["pamples","wamples"], "host":"localhost", "plugin":"metric", "type":"type0"}]`,
	"Multiple Metrics":          `[{"values": [1234], "dstypes": ["derive"], "dsnames":["samples"], "host":"localhost", "plugin":"metric", "type":"type0"}, {"values": [5678], "dstypes": ["derive"], "dsnames":["samples"], "host":"localhost", "plugin":"metric", "type":"type1"}]`,
}

var expected = `
{
    "Without Instance Types": [
        {
          "Name": "collectd_metric_type0_samples_total",
          "LabelKeys": [
            "host",
            "plugin_instance",
            "type_instance"
          ],
          "LabelVals": [
            "localhost",
            "base",
            "base"
          ],
          "Time": 0,
          "Type": 1,
          "Interval": 0,
          "Value": 2121
        }
      ],
    "With Instance Types": [
        {
          "Name": "collectd_metric_type666_samples_total",
          "LabelKeys": [
            "host",
            "plugin_instance",
            "type_instance"
          ],
          "LabelVals": [
            "localhost",
            "plugin0",
            "type0"
          ],
          "Time": 0,
          "Type": 1,
          "Interval": 0,
          "Value": 2122
        }
      ],
    "Multi-dimensional Metrics": [
        {
          "Name": "collectd_metric_type0_pamples_total",
          "LabelKeys": [
            "host",
            "plugin_instance",
            "type_instance"
          ],
          "LabelVals": [
            "localhost",
            "base",
            "base"
          ],
          "Time": 0,
          "Type": 1,
          "Interval": 0,
          "Value": 2112
        },
        {
          "Name": "collectd_metric_type0_wamples_total",
          "LabelKeys": [
            "host",
            "plugin_instance",
            "type_instance"
          ],
          "LabelVals": [
            "localhost",
            "base",
            "base"
          ],
          "Time": 0,
          "Type": 1,
          "Interval": 0,
          "Value": 1001
        }
      ],
    "Multiple Metrics": [
        {
          "Name": "collectd_metric_type0_samples_total",
          "LabelKeys": [
            "host",
            "plugin_instance",
            "type_instance"
          ],
          "LabelVals": [
            "localhost",
            "base",
            "base"
          ],
          "Time": 0,
          "Type": 1,
          "Interval": 0,
          "Value": 1234
        },
        {
          "Name": "collectd_metric_type1_samples_total",
          "LabelKeys": [
            "host",
            "plugin_instance",
            "type_instance"
          ],
          "LabelVals": [
            "localhost",
            "base",
            "base"
          ],
          "Time": 0,
          "Type": 1,
          "Interval": 0,
          "Value": 5678
        }
      ]
}
`

var (
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

// Use this to update messages in metric-tests-expected.json if behavior should change

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
	validResults := map[string][]data.Metric{}
	err := json.Unmarshal([]byte(expected), &validResults)
	if err != nil {
		t.Error(err)
	}

	metricHandler := New().(*collectdMetricsHandler)
	t.Run("Invalid Messages", func(t *testing.T) {
		for _, blob := range testMsgsInvalid {
			metricHandler.totalDecodeErrors = 0
			_ = metricHandler.Handle([]byte(blob), false, MetricReceive, EventReceive)
			assert.Equal(t, uint64(1), metricHandler.totalDecodeErrors)
		}
	})

	metricHandler.totalDecodeErrors = 0
	t.Run("Valid Messages", func(t *testing.T) {
		for test, blob := range testMsgsValid {
			metricsUT = []data.Metric{}
			err := metricHandler.Handle([]byte(blob), false, MetricReceive, EventReceive)
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, uint64(0), metricHandler.totalDecodeErrors)
			assert.ElementsMatchf(t, validResults[test], metricsUT, "Failed: %s", test)
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
