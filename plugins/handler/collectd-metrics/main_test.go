package main

import (
	"github.com/infrawatch/sg-core/pkg/data"
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

var validResults map[string][]data.Metric = map[string][]data.Metric{
	"Without Instance Types": {{
		Name:  "collectd_metric_type0_samples_total",
		Value: 2121.0,
		Type:  data.COUNTER,
		Labels: map[string]string{
			"host":            "localhost",
			"plugin_instance": "base",
			"type_instance":   "base",
		},
	}},
	"With Instance Types": {{
		Name:  "collectd_metric_type0_samples_total",
		Value: 2121.0,
		Type:  data.COUNTER,
		Labels: map[string]string{
			"host":            "localhost",
			"plugin_instance": "plugin0",
			"type_instance":   "type0",
		},
	}},
	"Multi-dimensional Metrics": {{
		Name:  "collectd_metric_type0_samples_total",
		Value: 2121.0,
		Type:  data.COUNTER,
		Labels: map[string]string{
			"host":            "localhost",
			"plugin_instance": "base",
			"type_instance":   "base",
		},
	}, {
		Name:  "collectd_metric_type0_samples_total",
		Value: 2121.0,
		Type:  data.COUNTER,
		Labels: map[string]string{
			"host":            "localhost",
			"plugin_instance": "base",
			"type_instance":   "base",
		},
	}, {
		Name:  "collectd_metric_type0_samples_total",
		Value: 1010.0,
		Type:  data.COUNTER,
		Labels: map[string]string{
			"host":            "localhost",
			"plugin_instance": "base",
			"type_instance":   "base",
		},
	}},
}

//TestMsgParsing collectd metric parsing
// func TestMsgParsing(t *testing.T) {
// 	metricHandler := New().(*collectdMetricsHandler)
// 	t.Run("Invalid Messages", func(t *testing.T) {
// 		for test, blob := range testMsgsInvalid {
// 			metricHandler.totalDecodeErrors = 0
// 			metricHandler.Handle([]byte(blob))
// 			assert.Equal(t, uint64(1), metricHandler.totalDecodeErrors, fmt.Sprintf("Wrong # of errors in test iteration '%s'", test))
// 		}
// 	})

// 	metricHandler.totalDecodeErrors = 0
// 	t.Run("Valid Messages", func(t *testing.T) {
// 		for test, blob := range testMsgsValid {
// 			metrics := metricHandler.Handle([]byte(blob))
// 			assert.Equal(t, uint64(0), metricHandler.totalDecodeErrors, test)

// 			assert.Equal(t, validResults[test], metrics[:len(validResults[test])], test)
// 		}
// 	})
// }

// func BenchmarkParsing(b *testing.B) {
// 	// GOMAXPROCS = 8
// 	// On thinkpad T480s, performs at ~ 195k m/s
// 	metricHandler := New().(*collectdMetricsHandler)
// 	for i := 0; i < b.N; i++ {
// 		metricHandler.Handle([]byte(testMsgsValid["Multi-dimensional Metrics"]))
// 	}
// }
