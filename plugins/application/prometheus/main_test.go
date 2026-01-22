package main

import (
	"context"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "prometheus_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	app := New(logger, nil)
	require.NotNil(t, app)

	prom, ok := app.(*Prometheus)
	require.True(t, ok)
	require.NotNil(t, prom.logger)
	require.Equal(t, "127.0.0.1", prom.configuration.Host)
	require.Equal(t, 3000, prom.configuration.Port)
	require.Equal(t, 2, prom.configuration.ExpirationMultiple)
	require.NotNil(t, prom.collectorExpiryProc)
}

func TestConfig(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "prometheus_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	t.Run("valid config", func(t *testing.T) {
		app := New(logger, nil)
		prom := app.(*Prometheus)

		config := `
host: 0.0.0.0
port: 8080
withTimeStamp: true
expirationMultiple: 3
`
		err := prom.Config([]byte(config))
		require.NoError(t, err)
		assert.Equal(t, "0.0.0.0", prom.configuration.Host)
		assert.Equal(t, 8080, prom.configuration.Port)
		assert.Equal(t, true, prom.configuration.WithTimestamp)
		assert.Equal(t, 3, prom.configuration.ExpirationMultiple)
	})

	t.Run("invalid yaml config", func(t *testing.T) {
		app := New(logger, nil)
		prom := app.(*Prometheus)

		config := `
this is: not: valid: yaml
`
		err := prom.Config([]byte(config))
		require.Error(t, err)
	})

	t.Run("default port when not specified", func(t *testing.T) {
		app := New(logger, nil)
		prom := app.(*Prometheus)

		config := `
host: 0.0.0.0
`
		err := prom.Config([]byte(config))
		require.NoError(t, err)
		assert.Equal(t, 3000, prom.configuration.Port)
	})
}

func TestNewPromCollector(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "prometheus_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	lw := &logWrapper{l: logger, plugin: "test"}

	t.Run("create collector with timestamp", func(t *testing.T) {
		pc := NewPromCollector(lw, 2, true)
		require.NotNil(t, pc)
		assert.Equal(t, 2, pc.dimensions)
		assert.Equal(t, true, pc.withtimestamp)
		assert.NotNil(t, pc.logger)
	})

	t.Run("create collector without timestamp", func(t *testing.T) {
		pc := NewPromCollector(lw, 3, false)
		require.NotNil(t, pc)
		assert.Equal(t, 3, pc.dimensions)
		assert.Equal(t, false, pc.withtimestamp)
	})
}

func TestPromCollector_Dimensions(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "prometheus_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	lw := &logWrapper{l: logger, plugin: "test"}
	pc := NewPromCollector(lw, 5, false)

	assert.Equal(t, 5, pc.Dimensions())
}

func TestMetricExpiry(t *testing.T) {
	t.Run("keepAlive updates lastArrival", func(t *testing.T) {
		me := &metricExpiry{
			lastArrival: time.Now().Add(-1 * time.Hour),
		}

		oldTime := me.lastArrival
		time.Sleep(10 * time.Millisecond)
		me.keepAlive()

		assert.True(t, me.lastArrival.After(oldTime))
	})

	t.Run("Expired returns true when interval exceeded", func(t *testing.T) {
		me := &metricExpiry{
			lastArrival: time.Now().Add(-2 * time.Second),
		}

		assert.True(t, me.Expired(1*time.Second))
	})

	t.Run("Expired returns false when interval not exceeded", func(t *testing.T) {
		me := &metricExpiry{
			lastArrival: time.Now(),
		}

		assert.False(t, me.Expired(1*time.Second))
	})

	t.Run("Delete calls delete function", func(t *testing.T) {
		deleteCalled := false
		me := &metricExpiry{
			delete: func() bool {
				deleteCalled = true
				return true
			},
		}

		result := me.Delete()
		assert.True(t, result)
		assert.True(t, deleteCalled)
	})
}

func TestCollectorExpiry(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "prometheus_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	lw := &logWrapper{l: logger, plugin: "test"}

	t.Run("Expired returns true when collector is empty", func(t *testing.T) {
		pc := NewPromCollector(lw, 2, false)
		ce := &collectorExpiry{
			collector: pc,
		}

		assert.True(t, ce.Expired(1*time.Second))
	})

	t.Run("Expired returns false when collector has metrics", func(t *testing.T) {
		pc := NewPromCollector(lw, 2, false)
		// Add a metric to the collector
		pc.mProc.Store("test", &metricProcess{})

		ce := &collectorExpiry{
			collector: pc,
		}

		assert.False(t, ce.Expired(1*time.Second))
	})

	t.Run("Delete calls delete function", func(t *testing.T) {
		pc := NewPromCollector(lw, 2, false)
		deleteCalled := false

		ce := &collectorExpiry{
			collector: pc,
			delete: func() bool {
				deleteCalled = true
				return true
			},
		}

		result := ce.Delete()
		assert.True(t, result)
		assert.True(t, deleteCalled)
	})
}

func TestSyncMapLen(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		var m sync.Map
		assert.Equal(t, 0, syncMapLen(&m))
	})

	t.Run("map with items", func(t *testing.T) {
		var m sync.Map
		m.Store("key1", "value1")
		m.Store("key2", "value2")
		m.Store("key3", "value3")
		assert.Equal(t, 3, syncMapLen(&m))
	})
}

func TestPromCollector_UpdateMetrics(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "prometheus_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	lw := &logWrapper{l: logger, plugin: "test"}

	t.Run("add new metric", func(t *testing.T) {
		pc := NewPromCollector(lw, 2, false)
		ep := newExpiryProc(10 * time.Second)

		pc.UpdateMetrics(
			"test_metric",
			123.456,
			data.GAUGE,
			5*time.Second,
			42.0,
			[]string{"label1", "label2"},
			[]string{"value1", "value2"},
			ep,
		)

		assert.Equal(t, 1, syncMapLen(&pc.mProc))

		// Verify the metric was stored correctly
		key := "test_metricvalue1value2"
		mProcItf, found := pc.mProc.Load(key)
		require.True(t, found)

		mProc := mProcItf.(*metricProcess)
		assert.Equal(t, "test_metric", mProc.metric.Name)
		assert.Equal(t, 42.0, mProc.metric.Value)
		assert.Equal(t, data.GAUGE, mProc.metric.Type)
		assert.Equal(t, 5*time.Second, mProc.metric.Interval)
		assert.Equal(t, []string{"label1", "label2"}, mProc.metric.LabelKeys)
		assert.Equal(t, []string{"value1", "value2"}, mProc.metric.LabelVals)
		assert.Equal(t, 123.456, mProc.metric.Time)
	})

	t.Run("update existing metric", func(t *testing.T) {
		pc := NewPromCollector(lw, 2, false)
		ep := newExpiryProc(10 * time.Second)

		// Add initial metric
		pc.UpdateMetrics(
			"test_metric",
			123.0,
			data.GAUGE,
			5*time.Second,
			42.0,
			[]string{"label1", "label2"},
			[]string{"value1", "value2"},
			ep,
		)

		// Update the same metric
		pc.UpdateMetrics(
			"test_metric",
			124.0,
			data.GAUGE,
			5*time.Second,
			99.0,
			[]string{"label1", "label2"},
			[]string{"value1", "value2"},
			ep,
		)

		// Should still have only one metric
		assert.Equal(t, 1, syncMapLen(&pc.mProc))

		// Verify the metric was updated
		key := "test_metricvalue1value2"
		mProcItf, found := pc.mProc.Load(key)
		require.True(t, found)

		mProc := mProcItf.(*metricProcess)
		assert.Equal(t, 99.0, mProc.metric.Value)
		assert.Equal(t, 124.0, mProc.metric.Time)
	})

	t.Run("multiple metrics with different label values", func(t *testing.T) {
		pc := NewPromCollector(lw, 2, false)
		ep := newExpiryProc(10 * time.Second)

		pc.UpdateMetrics(
			"test_metric",
			123.0,
			data.GAUGE,
			5*time.Second,
			42.0,
			[]string{"label1", "label2"},
			[]string{"value1", "value2"},
			ep,
		)

		pc.UpdateMetrics(
			"test_metric",
			123.0,
			data.GAUGE,
			5*time.Second,
			43.0,
			[]string{"label1", "label2"},
			[]string{"value1", "value3"},
			ep,
		)

		// Should have two different metrics
		assert.Equal(t, 2, syncMapLen(&pc.mProc))
	})
}

func TestPromCollector_Describe(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "prometheus_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	lw := &logWrapper{l: logger, plugin: "test"}
	pc := NewPromCollector(lw, 2, false)
	ep := newExpiryProc(10 * time.Second)

	// Add some metrics
	pc.UpdateMetrics(
		"metric1",
		123.0,
		data.GAUGE,
		5*time.Second,
		42.0,
		[]string{"label1"},
		[]string{"value1"},
		ep,
	)

	pc.UpdateMetrics(
		"metric2",
		124.0,
		data.COUNTER,
		5*time.Second,
		43.0,
		[]string{"label1"},
		[]string{"value2"},
		ep,
	)

	ch := make(chan *prometheus.Desc, 10)
	go func() {
		pc.Describe(ch)
		close(ch)
	}()

	descriptions := []string{}
	for desc := range ch {
		descriptions = append(descriptions, desc.String())
	}

	assert.Equal(t, 2, len(descriptions))
}

func TestPromCollector_Collect(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "prometheus_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	lw := &logWrapper{l: logger, plugin: "test"}

	t.Run("collect without timestamp", func(t *testing.T) {
		pc := NewPromCollector(lw, 1, false)
		ep := newExpiryProc(10 * time.Second)

		pc.UpdateMetrics(
			"test_metric",
			123.0,
			data.GAUGE,
			5*time.Second,
			42.0,
			[]string{"label1"},
			[]string{"value1"},
			ep,
		)

		ch := make(chan prometheus.Metric, 10)
		go func() {
			pc.Collect(ch)
			close(ch)
		}()

		metrics := []prometheus.Metric{}
		for metric := range ch {
			metrics = append(metrics, metric)
		}

		assert.Equal(t, 1, len(metrics))
	})

	t.Run("collect with timestamp", func(t *testing.T) {
		pc := NewPromCollector(lw, 1, true)
		ep := newExpiryProc(10 * time.Second)

		pc.UpdateMetrics(
			"test_metric",
			123.0,
			data.GAUGE,
			5*time.Second,
			42.0,
			[]string{"label1"},
			[]string{"value1"},
			ep,
		)

		ch := make(chan prometheus.Metric, 10)
		go func() {
			pc.Collect(ch)
			close(ch)
		}()

		metrics := []prometheus.Metric{}
		for metric := range ch {
			metrics = append(metrics, metric)
		}

		assert.Equal(t, 1, len(metrics))

		// Verify that metric has a timestamp
		var m dto.Metric
		err := metrics[0].Write(&m)
		require.NoError(t, err)
		assert.NotNil(t, m.TimestampMs, "metric should have a timestamp")
		assert.Equal(t, int64(123000), *m.TimestampMs, "timestamp should be 123 seconds in milliseconds")
	})

	t.Run("collect with zero timestamp", func(t *testing.T) {
		pc := NewPromCollector(lw, 1, true)
		ep := newExpiryProc(10 * time.Second)

		pc.UpdateMetrics(
			"test_metric",
			0.0, // zero timestamp
			data.GAUGE,
			5*time.Second,
			42.0,
			[]string{"label1"},
			[]string{"value1"},
			ep,
		)

		ch := make(chan prometheus.Metric, 10)
		go func() {
			pc.Collect(ch)
			close(ch)
		}()

		metrics := []prometheus.Metric{}
		for metric := range ch {
			metrics = append(metrics, metric)
		}

		assert.Equal(t, 1, len(metrics))

		// Verify that metric does NOT have a timestamp when zero timestamp is provided
		var m dto.Metric
		err := metrics[0].Write(&m)
		require.NoError(t, err)
		assert.Nil(t, m.TimestampMs, "metric should not have a timestamp when zero timestamp is provided")
	})

	t.Run("collect marks metrics as scrapped", func(t *testing.T) {
		pc := NewPromCollector(lw, 1, false)
		ep := newExpiryProc(10 * time.Second)

		pc.UpdateMetrics(
			"test_metric",
			123.0,
			data.GAUGE,
			5*time.Second,
			42.0,
			[]string{"label1"},
			[]string{"value1"},
			ep,
		)

		key := "test_metricvalue1"
		mProcItf, found := pc.mProc.Load(key)
		require.True(t, found)
		mProc := mProcItf.(*metricProcess)
		assert.False(t, mProc.scrapped)

		ch := make(chan prometheus.Metric, 10)
		go func() {
			pc.Collect(ch)
			close(ch)
		}()

		for range ch {
			// Drain channel
		}

		// Check that scrapped flag is set
		mProcItf, found = pc.mProc.Load(key)
		require.True(t, found)
		mProc = mProcItf.(*metricProcess)
		assert.True(t, mProc.scrapped)
	})
}

func TestReceiveMetric(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "prometheus_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	t.Run("receive metric creates collector", func(t *testing.T) {
		app := New(logger, nil)
		prom := app.(*Prometheus)
		prom.ctx = context.Background()
		prom.registry = prometheus.NewRegistry()

		prom.ReceiveMetric(
			"test_metric",
			123.0,
			data.GAUGE,
			5*time.Second,
			42.0,
			[]string{"label1"},
			[]string{"value1"},
		)

		// Should have created a collector with dimension 1
		assert.Equal(t, 1, syncMapLen(&prom.collectors))
	})

	t.Run("receive multiple metrics with same dimensions", func(t *testing.T) {
		app := New(logger, nil)
		prom := app.(*Prometheus)
		prom.ctx = context.Background()
		prom.registry = prometheus.NewRegistry()

		prom.ReceiveMetric(
			"metric1",
			123.0,
			data.GAUGE,
			5*time.Second,
			42.0,
			[]string{"label1"},
			[]string{"value1"},
		)

		prom.ReceiveMetric(
			"metric2",
			124.0,
			data.COUNTER,
			5*time.Second,
			43.0,
			[]string{"label2"},
			[]string{"value2"},
		)

		// Should still have only one collector (both have 1 dimension)
		assert.Equal(t, 1, syncMapLen(&prom.collectors))
	})

	t.Run("receive metrics with different dimensions", func(t *testing.T) {
		app := New(logger, nil)
		prom := app.(*Prometheus)
		prom.ctx = context.Background()
		prom.registry = prometheus.NewRegistry()

		prom.ReceiveMetric(
			"metric1",
			123.0,
			data.GAUGE,
			5*time.Second,
			42.0,
			[]string{"label1"},
			[]string{"value1"},
		)

		prom.ReceiveMetric(
			"metric2",
			124.0,
			data.COUNTER,
			5*time.Second,
			43.0,
			[]string{"label1", "label2"},
			[]string{"value1", "value2"},
		)

		// Should have two collectors (dimensions 1 and 2)
		assert.Equal(t, 2, syncMapLen(&prom.collectors))
	})

	t.Run("receive metric creates expiry process", func(t *testing.T) {
		app := New(logger, nil)
		prom := app.(*Prometheus)
		prom.ctx = context.Background()
		prom.registry = prometheus.NewRegistry()

		prom.ReceiveMetric(
			"test_metric",
			123.0,
			data.GAUGE,
			5*time.Second,
			42.0,
			[]string{"label1"},
			[]string{"value1"},
		)

		// Should have created an expiry process for 5s interval
		assert.Equal(t, 1, syncMapLen(&prom.metricExpiryProcs))
	})

	t.Run("receive metrics with different intervals", func(t *testing.T) {
		app := New(logger, nil)
		prom := app.(*Prometheus)
		prom.ctx = context.Background()
		prom.registry = prometheus.NewRegistry()

		prom.ReceiveMetric(
			"metric1",
			123.0,
			data.GAUGE,
			5*time.Second,
			42.0,
			[]string{"label1"},
			[]string{"value1"},
		)

		prom.ReceiveMetric(
			"metric2",
			124.0,
			data.COUNTER,
			10*time.Second,
			43.0,
			[]string{"label1"},
			[]string{"value2"},
		)

		// Should have two expiry processes (5s and 10s)
		assert.Equal(t, 2, syncMapLen(&prom.metricExpiryProcs))
	})
}
