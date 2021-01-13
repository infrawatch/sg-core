package main

import (
	"time"

	"github.com/go-openapi/errors"
	"github.com/infrawatch/sg-core-refactor/pkg/data"
	"github.com/infrawatch/sg-core-refactor/pkg/handler"
	"github.com/infrawatch/sg-core-refactor/plugins/handler/collectd-metrics/pkg/collectd"
)

type collectdMetricsHandler struct {
	totalMetricsReceived uint64 //total number of internal metrics created from collectd blobs
	totalDecodeErrors    uint64
}

func (c *collectdMetricsHandler) Handle(blob []byte) []data.Metric {

	var err error
	var cdmetrics *[]collectd.Metric

	cdmetrics, err = collectd.ParseInputByte(blob)
	metrics := []data.Metric{}

	if err != nil {
		c.totalDecodeErrors++
		return metrics
	}

	var ms []data.Metric
	if cdmetrics == nil {
		c.totalDecodeErrors++
		return metrics
	}

	for _, cdmetric := range *cdmetrics {
		ms, err = c.createMetrics(&cdmetric)
		if err != nil {
			c.totalDecodeErrors++
		}
		metrics = append(metrics, ms...)
	}

	metrics = append(metrics, []data.Metric{{
		Name:  "sg_total_metric_rcv_count",
		Type:  data.COUNTER,
		Value: float64(c.totalMetricsReceived),
		Time:  time.Now(),
		Labels: map[string]string{
			"source": "SG",
		},
	}, {
		Name:  "sg_total_metric_decode_error_count",
		Type:  data.COUNTER,
		Value: float64(c.totalDecodeErrors),
		Time:  time.Now(),
		Labels: map[string]string{
			"source": "SG",
		},
	},
	}...)

	return metrics
}

func (c *collectdMetricsHandler) createMetrics(cdmetric *collectd.Metric) ([]data.Metric, error) {
	if !validateMetric(cdmetric) {
		return nil, errors.New(0, "")
	}
	pluginInstance := cdmetric.PluginInstance
	if pluginInstance == "" {
		pluginInstance = "base"
	}
	typeInstance := cdmetric.TypeInstance
	if typeInstance == "" {
		typeInstance = "base"
	}

	equal := int64((len(cdmetric.Dsnames) ^ len(cdmetric.Dstypes)) ^ (len(cdmetric.Dsnames) ^ len(cdmetric.Values)))
	if equal != 0 {
		return nil, errors.New(0, "")
	}

	var metrics []data.Metric
	for index := range cdmetric.Dsnames {
		metrics = append(metrics,
			data.Metric{
				Name:  genMetricName(cdmetric, index),
				Type:  strToMetricType(cdmetric.Dstypes[index]),
				Value: cdmetric.Values[index],
				Time:  cdmetric.Time.Time(),
				Labels: map[string]string{
					"host":            cdmetric.Host,
					"plugin_instance": pluginInstance,
					"type_instance":   typeInstance,
				}})
		c.totalMetricsReceived++
	}
	return metrics, nil
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

func strToMetricType(msg string) data.MetricType {
	if mt, ok := map[string]data.MetricType{
		"counter":  data.COUNTER,
		"absolute": data.UNTYPED,
		"derive":   data.COUNTER,
		"gauge":    data.GAUGE,
	}[msg]; ok {
		return mt
	}
	return data.UNTYPED
}

//New create new collectdMetricsHandler object
func New() handler.MetricHandler {
	return &collectdMetricsHandler{}
}
