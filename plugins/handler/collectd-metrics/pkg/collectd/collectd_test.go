package collectd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInputByte(t *testing.T) {
	t.Run("parse valid single metric", func(t *testing.T) {
		input := []byte(`[{
			"values": [2121],
			"dstypes": ["derive"],
			"dsnames": ["samples"],
			"time": 1234567890,
			"interval": 10,
			"host": "localhost",
			"plugin": "cpu",
			"plugin_instance": "0",
			"type": "cpu",
			"type_instance": "idle"
		}]`)

		metrics, err := ParseInputByte(input)
		require.NoError(t, err)
		require.NotNil(t, metrics)
		require.Equal(t, 1, len(*metrics))

		metric := (*metrics)[0]
		assert.Equal(t, []float64{2121}, metric.Values)
		assert.Equal(t, []string{"derive"}, metric.Dstypes)
		assert.Equal(t, []string{"samples"}, metric.Dsnames)
		assert.Equal(t, float64(10), metric.Interval)
		assert.Equal(t, "localhost", metric.Host)
		assert.Equal(t, "cpu", metric.Plugin)
		assert.Equal(t, "0", metric.PluginInstance)
		assert.Equal(t, "cpu", metric.Type)
		assert.Equal(t, "idle", metric.TypeInstance)
	})

	t.Run("parse multiple metrics", func(t *testing.T) {
		input := []byte(`[
			{
				"values": [100],
				"dstypes": ["derive"],
				"dsnames": ["rx"],
				"host": "host1",
				"plugin": "interface",
				"type": "if_octets"
			},
			{
				"values": [200],
				"dstypes": ["derive"],
				"dsnames": ["tx"],
				"host": "host1",
				"plugin": "interface",
				"type": "if_octets"
			},
			{
				"values": [50],
				"dstypes": ["gauge"],
				"dsnames": ["value"],
				"host": "host2",
				"plugin": "cpu",
				"type": "percent"
			}
		]`)

		metrics, err := ParseInputByte(input)
		require.NoError(t, err)
		require.NotNil(t, metrics)
		require.Equal(t, 3, len(*metrics))

		assert.Equal(t, "interface", (*metrics)[0].Plugin)
		assert.Equal(t, "interface", (*metrics)[1].Plugin)
		assert.Equal(t, "cpu", (*metrics)[2].Plugin)
		assert.Equal(t, float64(100), (*metrics)[0].Values[0])
		assert.Equal(t, float64(200), (*metrics)[1].Values[0])
		assert.Equal(t, float64(50), (*metrics)[2].Values[0])
	})

	t.Run("parse multi-dimensional metric", func(t *testing.T) {
		input := []byte(`[{
			"values": [2112, 1001, 5555],
			"dstypes": ["derive", "counter", "gauge"],
			"dsnames": ["rx", "tx", "errors"],
			"host": "localhost",
			"plugin": "virt",
			"type": "if_packets"
		}]`)

		metrics, err := ParseInputByte(input)
		require.NoError(t, err)
		require.NotNil(t, metrics)
		require.Equal(t, 1, len(*metrics))

		metric := (*metrics)[0]
		assert.Equal(t, 3, len(metric.Values))
		assert.Equal(t, []float64{2112, 1001, 5555}, metric.Values)
		assert.Equal(t, []string{"derive", "counter", "gauge"}, metric.Dstypes)
		assert.Equal(t, []string{"rx", "tx", "errors"}, metric.Dsnames)
	})

	t.Run("parse metric without optional fields", func(t *testing.T) {
		input := []byte(`[{
			"values": [42],
			"dstypes": ["gauge"],
			"dsnames": ["value"],
			"host": "localhost",
			"plugin": "memory",
			"type": "memory"
		}]`)

		metrics, err := ParseInputByte(input)
		require.NoError(t, err)
		require.NotNil(t, metrics)
		require.Equal(t, 1, len(*metrics))

		metric := (*metrics)[0]
		assert.Equal(t, "", metric.PluginInstance)
		assert.Equal(t, "", metric.TypeInstance)
	})

	t.Run("parse metric with time and interval", func(t *testing.T) {
		input := []byte(`[{
			"values": [100],
			"dstypes": ["derive"],
			"dsnames": ["samples"],
			"time": 1609459200.5,
			"interval": 10.5,
			"host": "localhost",
			"plugin": "cpu",
			"type": "cpu"
		}]`)

		metrics, err := ParseInputByte(input)
		require.NoError(t, err)
		require.NotNil(t, metrics)
		require.Equal(t, 1, len(*metrics))

		metric := (*metrics)[0]
		assert.NotNil(t, metric.Time)
		assert.Equal(t, float64(10.5), metric.Interval)
	})

	t.Run("parse metric with metadata", func(t *testing.T) {
		input := []byte(`[{
			"values": [100],
			"dstypes": ["derive"],
			"dsnames": ["samples"],
			"host": "localhost",
			"plugin": "cpu",
			"type": "cpu",
			"meta": {
				"key1": "value1",
				"key2": 123
			}
		}]`)

		metrics, err := ParseInputByte(input)
		require.NoError(t, err)
		require.NotNil(t, metrics)
		require.Equal(t, 1, len(*metrics))
	})

	t.Run("error on invalid JSON", func(t *testing.T) {
		input := []byte(`{invalid json}`)

		metrics, err := ParseInputByte(input)
		require.Error(t, err)
		assert.Nil(t, metrics)
	})

	t.Run("error on non-array JSON", func(t *testing.T) {
		input := []byte(`{"values": [100]}`)

		metrics, err := ParseInputByte(input)
		require.Error(t, err)
		assert.Nil(t, metrics)
	})

	t.Run("parse empty array", func(t *testing.T) {
		input := []byte(`[]`)

		metrics, err := ParseInputByte(input)
		require.NoError(t, err)
		require.NotNil(t, metrics)
		assert.Equal(t, 0, len(*metrics))
	})

	t.Run("parse metric with all dstype variations", func(t *testing.T) {
		input := []byte(`[{
			"values": [1, 2, 3, 4],
			"dstypes": ["derive", "counter", "gauge", "absolute"],
			"dsnames": ["d1", "d2", "d3", "d4"],
			"host": "localhost",
			"plugin": "test",
			"type": "test"
		}]`)

		metrics, err := ParseInputByte(input)
		require.NoError(t, err)
		require.NotNil(t, metrics)
		require.Equal(t, 1, len(*metrics))

		metric := (*metrics)[0]
		assert.Equal(t, 4, len(metric.Values))
		assert.Equal(t, []string{"derive", "counter", "gauge", "absolute"}, metric.Dstypes)
	})

	t.Run("parse real-world virt plugin data", func(t *testing.T) {
		input := []byte(`[
			{
				"values": [1234.5, 5678.9],
				"dstypes": ["derive", "counter"],
				"dsnames": ["rx", "tx"],
				"host": "controller-0.redhat.local",
				"time": 1609459200,
				"interval": 5,
				"plugin": "virt",
				"plugin_instance": "instance-00000001",
				"type": "if_packets",
				"type_instance": "tap73125d-60"
			}
		]`)

		metrics, err := ParseInputByte(input)
		require.NoError(t, err)
		require.NotNil(t, metrics)
		require.Equal(t, 1, len(*metrics))

		metric := (*metrics)[0]
		assert.Equal(t, 2, len(metric.Values))
		assert.Equal(t, "controller-0.redhat.local", metric.Host)
		assert.Equal(t, "virt", metric.Plugin)
		assert.Equal(t, "instance-00000001", metric.PluginInstance)
		assert.Equal(t, "if_packets", metric.Type)
		assert.Equal(t, "tap73125d-60", metric.TypeInstance)
		assert.Equal(t, []string{"rx", "tx"}, metric.Dsnames)
	})

	t.Run("parse metric with floating point values", func(t *testing.T) {
		input := []byte(`[{
			"values": [123.456, 789.012],
			"dstypes": ["gauge", "gauge"],
			"dsnames": ["value1", "value2"],
			"host": "localhost",
			"plugin": "cpu",
			"type": "percent"
		}]`)

		metrics, err := ParseInputByte(input)
		require.NoError(t, err)
		require.NotNil(t, metrics)
		require.Equal(t, 1, len(*metrics))

		metric := (*metrics)[0]
		assert.InDelta(t, 123.456, metric.Values[0], 0.001)
		assert.InDelta(t, 789.012, metric.Values[1], 0.001)
	})

	t.Run("parse metric with zero values", func(t *testing.T) {
		input := []byte(`[{
			"values": [0, 0, 0],
			"dstypes": ["derive", "derive", "derive"],
			"dsnames": ["a", "b", "c"],
			"host": "localhost",
			"plugin": "test",
			"type": "test"
		}]`)

		metrics, err := ParseInputByte(input)
		require.NoError(t, err)
		require.NotNil(t, metrics)
		require.Equal(t, 1, len(*metrics))

		metric := (*metrics)[0]
		assert.Equal(t, []float64{0, 0, 0}, metric.Values)
	})

	t.Run("parse metric with negative values", func(t *testing.T) {
		input := []byte(`[{
			"values": [-100, -50.5],
			"dstypes": ["gauge", "gauge"],
			"dsnames": ["temp", "pressure"],
			"host": "localhost",
			"plugin": "sensors",
			"type": "temperature"
		}]`)

		metrics, err := ParseInputByte(input)
		require.NoError(t, err)
		require.NotNil(t, metrics)
		require.Equal(t, 1, len(*metrics))

		metric := (*metrics)[0]
		assert.Equal(t, float64(-100), metric.Values[0])
		assert.Equal(t, float64(-50.5), metric.Values[1])
	})

	t.Run("parse metric with very large values", func(t *testing.T) {
		input := []byte(`[{
			"values": [9999999999999, 1234567890123],
			"dstypes": ["counter", "counter"],
			"dsnames": ["bytes_in", "bytes_out"],
			"host": "localhost",
			"plugin": "interface",
			"type": "if_octets"
		}]`)

		metrics, err := ParseInputByte(input)
		require.NoError(t, err)
		require.NotNil(t, metrics)
		require.Equal(t, 1, len(*metrics))

		metric := (*metrics)[0]
		assert.Equal(t, float64(9999999999999), metric.Values[0])
		assert.Equal(t, float64(1234567890123), metric.Values[1])
	})

	t.Run("parse metric with special characters in strings", func(t *testing.T) {
		input := []byte(`[{
			"values": [100],
			"dstypes": ["gauge"],
			"dsnames": ["value"],
			"host": "host-name.with-dashes",
			"plugin": "plugin_with_underscores",
			"plugin_instance": "instance.0",
			"type": "type-name",
			"type_instance": "instance_name"
		}]`)

		metrics, err := ParseInputByte(input)
		require.NoError(t, err)
		require.NotNil(t, metrics)
		require.Equal(t, 1, len(*metrics))

		metric := (*metrics)[0]
		assert.Equal(t, "host-name.with-dashes", metric.Host)
		assert.Equal(t, "plugin_with_underscores", metric.Plugin)
		assert.Equal(t, "instance.0", metric.PluginInstance)
		assert.Equal(t, "type-name", metric.Type)
		assert.Equal(t, "instance_name", metric.TypeInstance)
	})
}
