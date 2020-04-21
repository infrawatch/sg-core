package unixserver

import (
	"testing"
	"time"

	"github.com/infrawatch/sg2/pkg/assert"
	"github.com/infrawatch/sg2/pkg/collectd"
	"github.com/prometheus/client_golang/prometheus"
)

var cd *collectd.Collectd

func TestStaleMetric(t *testing.T) {
	cd := &collectd.Collectd{
		Values:  []float64{1.59},
		Host:    "localhost",
		Dstypes: []string{"gauge"},
		Dsnames: []string{"dsname0"},
		Plugin:  "interface",
		Type:    "ingress",
	}

	cdmetrics := NewCDMetrics()
	ch := make(chan prometheus.Metric)

	MAXTTL = 1

	cdmetrics.updateOrAddMetrics(cd)
	t.Log(cdmetrics.metrics)
	assert.Equals(t, 1, len(cdmetrics.metrics))
	for i := 0; i < 2; i++ {
		go cdmetrics.Collect(ch)
		time.Sleep(time.Second * 1)
	}

	assert.Equals(t, 0, len(cdmetrics.metrics))
}
