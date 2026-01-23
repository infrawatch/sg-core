package data

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMetricType_String(t *testing.T) {
	t.Run("UNTYPED metric type", func(t *testing.T) {
		mt := UNTYPED
		assert.Equal(t, "untyped", mt.String())
	})

	t.Run("COUNTER metric type", func(t *testing.T) {
		mt := COUNTER
		assert.Equal(t, "counter", mt.String())
	})

	t.Run("GAUGE metric type", func(t *testing.T) {
		mt := GAUGE
		assert.Equal(t, "gauge", mt.String())
	})

	t.Run("metric type with value 0", func(t *testing.T) {
		mt := MetricType(0)
		assert.Equal(t, "untyped", mt.String())
	})

	t.Run("metric type with value 1", func(t *testing.T) {
		mt := MetricType(1)
		assert.Equal(t, "counter", mt.String())
	})

	t.Run("metric type with value 2", func(t *testing.T) {
		mt := MetricType(2)
		assert.Equal(t, "gauge", mt.String())
	})
}

func TestEventType_String(t *testing.T) {
	t.Run("ERROR event type", func(t *testing.T) {
		et := ERROR
		assert.Equal(t, "error", et.String())
	})

	t.Run("EVENT event type", func(t *testing.T) {
		et := EVENT
		assert.Equal(t, "event", et.String())
	})

	t.Run("LOG event type", func(t *testing.T) {
		et := LOG
		assert.Equal(t, "log", et.String())
	})

	t.Run("RESULT event type", func(t *testing.T) {
		et := RESULT
		assert.Equal(t, "result", et.String())
	})

	t.Run("TASK event type", func(t *testing.T) {
		et := TASK
		assert.Equal(t, "task", et.String())
	})

	t.Run("event type with value 0", func(t *testing.T) {
		et := EventType(0)
		assert.Equal(t, "error", et.String())
	})

	t.Run("event type with value 1", func(t *testing.T) {
		et := EventType(1)
		assert.Equal(t, "event", et.String())
	})

	t.Run("event type with value 2", func(t *testing.T) {
		et := EventType(2)
		assert.Equal(t, "log", et.String())
	})

	t.Run("event type with value 3", func(t *testing.T) {
		et := EventType(3)
		assert.Equal(t, "result", et.String())
	})

	t.Run("event type with value 4", func(t *testing.T) {
		et := EventType(4)
		assert.Equal(t, "task", et.String())
	})
}

func TestEventSeverity_String(t *testing.T) {
	t.Run("UNKNOWN severity", func(t *testing.T) {
		es := UNKNOWN
		assert.Equal(t, "unknown", es.String())
	})

	t.Run("DEBUG severity", func(t *testing.T) {
		es := DEBUG
		assert.Equal(t, "debug", es.String())
	})

	t.Run("INFO severity", func(t *testing.T) {
		es := INFO
		assert.Equal(t, "info", es.String())
	})

	t.Run("WARNING severity", func(t *testing.T) {
		es := WARNING
		assert.Equal(t, "warning", es.String())
	})

	t.Run("CRITICAL severity", func(t *testing.T) {
		es := CRITICAL
		assert.Equal(t, "critical", es.String())
	})

	t.Run("severity with value 0", func(t *testing.T) {
		es := EventSeverity(0)
		assert.Equal(t, "unknown", es.String())
	})

	t.Run("severity with value 1", func(t *testing.T) {
		es := EventSeverity(1)
		assert.Equal(t, "debug", es.String())
	})

	t.Run("severity with value 2", func(t *testing.T) {
		es := EventSeverity(2)
		assert.Equal(t, "info", es.String())
	})

	t.Run("severity with value 3", func(t *testing.T) {
		es := EventSeverity(3)
		assert.Equal(t, "warning", es.String())
	})

	t.Run("severity with value 4", func(t *testing.T) {
		es := EventSeverity(4)
		assert.Equal(t, "critical", es.String())
	})
}

func TestEvent(t *testing.T) {
	t.Run("create event with all fields", func(t *testing.T) {
		event := Event{
			Index:     "test-index",
			Time:      float64(time.Now().Unix()),
			Type:      EVENT,
			Publisher: "test-publisher",
			Severity:  INFO,
			Labels: map[string]interface{}{
				"host": "localhost",
				"pod":  "test-pod",
			},
			Annotations: map[string]interface{}{
				"description": "test event",
			},
			Message: "test message",
		}

		assert.Equal(t, "test-index", event.Index)
		assert.Equal(t, EVENT, event.Type)
		assert.Equal(t, "test-publisher", event.Publisher)
		assert.Equal(t, INFO, event.Severity)
		assert.Equal(t, "test message", event.Message)
		assert.NotNil(t, event.Labels)
		assert.NotNil(t, event.Annotations)
	})

	t.Run("create event with minimal fields", func(t *testing.T) {
		event := Event{
			Type:    ERROR,
			Message: "error message",
		}

		assert.Equal(t, ERROR, event.Type)
		assert.Equal(t, "error message", event.Message)
	})
}

func TestMetric(t *testing.T) {
	t.Run("create metric with all fields", func(t *testing.T) {
		metric := Metric{
			Name:      "cpu_usage",
			Time:      float64(time.Now().Unix()),
			Type:      GAUGE,
			Interval:  time.Second * 10,
			Value:     75.5,
			LabelKeys: []string{"host", "cpu"},
			LabelVals: []string{"localhost", "cpu0"},
		}

		assert.Equal(t, "cpu_usage", metric.Name)
		assert.Equal(t, GAUGE, metric.Type)
		assert.Equal(t, 75.5, metric.Value)
		assert.Equal(t, time.Second*10, metric.Interval)
		assert.Equal(t, []string{"host", "cpu"}, metric.LabelKeys)
		assert.Equal(t, []string{"localhost", "cpu0"}, metric.LabelVals)
	})

	t.Run("create counter metric", func(t *testing.T) {
		metric := Metric{
			Name:  "requests_total",
			Type:  COUNTER,
			Value: 1000,
		}

		assert.Equal(t, "requests_total", metric.Name)
		assert.Equal(t, COUNTER, metric.Type)
		assert.Equal(t, float64(1000), metric.Value)
	})

	t.Run("create untyped metric", func(t *testing.T) {
		metric := Metric{
			Name:  "unknown_metric",
			Type:  UNTYPED,
			Value: 42,
		}

		assert.Equal(t, "unknown_metric", metric.Name)
		assert.Equal(t, UNTYPED, metric.Type)
		assert.Equal(t, float64(42), metric.Value)
	})

	t.Run("create metric with empty labels", func(t *testing.T) {
		metric := Metric{
			Name:      "simple_metric",
			Type:      GAUGE,
			Value:     123.45,
			LabelKeys: []string{},
			LabelVals: []string{},
		}

		assert.Equal(t, "simple_metric", metric.Name)
		assert.Empty(t, metric.LabelKeys)
		assert.Empty(t, metric.LabelVals)
	})
}
