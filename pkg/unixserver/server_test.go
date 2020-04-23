package unixserver

import (
	"context"
	"testing"
	"time"

	"github.com/infrawatch/sg2/pkg/assert"
	"github.com/infrawatch/sg2/pkg/cacheutil"
	"github.com/infrawatch/sg2/pkg/collectd"
)

func TestCDMetrics(t *testing.T) {
	t.Run("CDMetrics expiration", func(t *testing.T) {
		cd := &collectd.Collectd{
			Values:   []float64{1.59},
			Host:     "localhost",
			Dstypes:  []string{"gauge"},
			Dsnames:  []string{"dsname0"},
			Plugin:   "interface",
			Type:     "ingress",
			Interval: 0.2, //expire happens at 5x this interval
		}

		cdmetrics := NewCDMetrics()
		// ch := make(chan prometheus.Metric)

		cs := cacheutil.NewCacheServer()
		ctx := context.Background()

		go cs.Run(ctx)

		cdmetrics.updateOrAddMetrics(cd, cs)
		assert.Equals(t, 1, len(cdmetrics.metrics))
		for i := 0; i < 3; i++ {
			// go cdmetrics.Collect(ch)
			time.Sleep(time.Second * 1)
		}

		assert.Equals(t, 0, len(cdmetrics.metrics))
	})
}
