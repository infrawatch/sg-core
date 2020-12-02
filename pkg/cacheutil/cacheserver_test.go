package cacheutil

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/infrawatch/sg-core/pkg/assert"
)

type deleteFn func()

type LabelSeries struct {
	interval    float64
	lastArrival time.Time
	deleteFn    deleteFn
}

func (ls *LabelSeries) Expired() bool {
	return time.Since(ls.lastArrival).Seconds() >= ls.interval
}

func (ls *LabelSeries) Delete() {
	ls.deleteFn()
}

type MetricStash struct {
	metrics map[string]map[string]*LabelSeries
}

func NewMetricStash() *MetricStash {
	return &MetricStash{
		metrics: make(map[string]map[string]*LabelSeries),
	}
}

func (ms *MetricStash) addMetric(metricName string, interval float64, numLabels int, cs *CacheServer) {
	for i := 0; i < numLabels; i++ {
		ls := LabelSeries{
			interval:    interval,
			lastArrival: time.Now(),
		}

		labelName := "test-label-" + strconv.Itoa(i)

		ls.deleteFn = func() {
			fmt.Printf("Label %s in metric %s deleted\n", labelName, metricName)
			delete(ms.metrics[metricName], labelName)

			if len(ms.metrics[metricName]) == 0 {
				delete(ms.metrics, metricName)
				fmt.Printf("Metrics %s deleted\n", metricName)
			}
		}

		if ms.metrics[metricName] == nil {
			ms.metrics[metricName] = make(map[string]*LabelSeries)
		}

		ms.metrics[metricName][labelName] = &ls
		cs.Register(&ls)
	}
}

func TestCacheExpiry(t *testing.T) {
	ms := NewMetricStash()

	cs := NewCacheServer()
	ctx := context.Background()

	// TODO: we should be capturing the error not ignoring it
	go func() {
		_ = cs.Run(ctx)
	}()

	t.Run("single entry", func(t *testing.T) {
		ms.addMetric("test-metric", 1, 1, cs)
		assert.Equals(t, 1, len(ms.metrics))
		for i := 0; i < 2; i++ {
			time.Sleep(time.Second * 1)
		}

		assert.Equals(t, 0, len(ms.metrics))
	})

	t.Run("different metrics and intervals", func(t *testing.T) {
		ms.addMetric("test-metric-1", 1, 1, cs)
		ms.addMetric("test-metric-2", 2, 1, cs)

		assert.Equals(t, 2, len(ms.metrics))
		for i := 0; i < 4; i++ {
			time.Sleep(time.Second * 1)
		}
		assert.Equals(t, 0, len(ms.metrics))
	})

	t.Run("multilabel metric", func(t *testing.T) {
		ms.addMetric("test-metric-1", 1, 10, cs)

		assert.Equals(t, 10, len(ms.metrics["test-metric-1"]))
		for i := 0; i < 2; i++ {
			time.Sleep(time.Second * 1)
		}
		assert.Equals(t, 0, len(ms.metrics))
	})
}
