package main

import (
	"time"

	"github.com/go-openapi/errors"
	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/handler"
	"github.com/infrawatch/sg-core/plugins/handler/collectd-metrics/pkg/collectd"
)

var (
	strToMetricType map[string]data.MetricType = map[string]data.MetricType{
		"counter":  data.COUNTER,
		"absolute": data.UNTYPED,
		"derive":   data.COUNTER,
		"gauge":    data.GAUGE,
	}
)

type collectdMetricsHandler struct {
	totalMetricsReceived uint64 //total number of internal metrics created from collectd blobs
	totalDecodeErrors    uint64
}

func (c *collectdMetricsHandler) Handle(blob []byte, pf bus.PublishFunc) {
	var err error
	var cdmetrics *[]collectd.Metric

	cdmetrics, err = collectd.ParseInputByte(blob)

	if err != nil {
		c.totalDecodeErrors++
	}

	if cdmetrics == nil {
		c.totalDecodeErrors++
	}

	for _, cdmetric := range *cdmetrics {
		err = c.writeMetrics(&cdmetric, pf)
		if err != nil {
			c.totalDecodeErrors++
		}
	}

	// metrics = append(metrics, []data.Metric{{
	// 	Name:     "sg_total_metric_rcv_count",
	// 	Type:     data.COUNTER,
	// 	Value:    float64(c.totalMetricsReceived),
	// 	Time:     time.Now(),
	// 	Interval: 0,
	// 	Labels: map[string]string{
	// 		"source": "SG",
	// 	},
	// }, {
	// 	Name:     "sg_total_metric_decode_error_count",
	// 	Type:     data.COUNTER,
	// 	Value:    float64(c.totalDecodeErrors),
	// 	Time:     time.Now(),
	// 	Interval: 0,
	// 	Labels: map[string]string{
	// 		"source": "SG",
	// 	},
	// },
	// }...)

}

func (c *collectdMetricsHandler) writeMetrics(cdmetric *collectd.Metric, pf bus.PublishFunc) error {
	if !validateMetric(cdmetric) {
		return errors.New(0, "")
	}
	pluginInstance := cdmetric.PluginInstance
	if pluginInstance == "" {
		pluginInstance = "base"
	}
	typeInstance := cdmetric.TypeInstance
	if typeInstance == "" {
		typeInstance = "base"
	}

	for index := range cdmetric.Dsnames {
		mType, found := strToMetricType[cdmetric.Dstypes[index]]
		if !found {
			mType = data.UNTYPED
		}
		pf(
			genMetricName(cdmetric, index),
			cdmetric.Time.Time(),
			mType,
			time.Duration(cdmetric.Interval)*time.Second,
			cdmetric.Values[index],
			[]string{"host", "plugin_instance", "type_instance"},
			[]string{cdmetric.Host, pluginInstance, typeInstance},
		)
		c.totalMetricsReceived++
	}
	return nil
}

func validateMetric(cdmetric *collectd.Metric) bool {
	if cdmetric.Dsnames == nil ||
		cdmetric.Dstypes == nil ||
		cdmetric.Values == nil ||
		cdmetric.Host == "" ||
		cdmetric.Plugin == "" ||
		cdmetric.Type == "" {
		return false
	}

	equal := int64((len(cdmetric.Dsnames) ^ len(cdmetric.Dstypes)) ^ (len(cdmetric.Dsnames) ^ len(cdmetric.Values)))
	if equal != 0 {
		return false
	}
	return true
}

func genMetricName(cdmetric *collectd.Metric, index int) (name string) {

	name = "collectd_" + cdmetric.Plugin + "_" + cdmetric.Type
	if cdmetric.Type == cdmetric.Plugin {
		name = "collectd_" + cdmetric.Plugin
	}

	if dsname := cdmetric.Dsnames[index]; dsname != "value" {
		name += "_" + dsname
	}

	switch cdmetric.Dstypes[index] {
	case "counter", "derive":
		name += "_total"
	}

	return
}

//New create new collectdMetricsHandler object
func New() handler.MetricHandler {
	return &collectdMetricsHandler{}
}
