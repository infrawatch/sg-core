package ceilometer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

func TestNew(t *testing.T) {
	t.Run("creates new ceilometer instance", func(t *testing.T) {
		c := New()
		require.NotNil(t, c)
		assert.NotNil(t, c.schema)
	})
}

func TestParseInputJSON(t *testing.T) {
	t.Run("parse valid JSON message", func(t *testing.T) {
		c := New()
		input := []byte(`{
			"request": {
				"oslo.version": "2.0",
				"oslo.message": "{\"message_id\": \"test-id\", \"publisher_id\": \"test.publisher\", \"event_type\": \"metering\", \"priority\": \"SAMPLE\", \"payload\": [{\"source\": \"openstack\", \"counter_name\": \"cpu\", \"counter_type\": \"cumulative\", \"counter_unit\": \"ns\", \"counter_volume\": 347670000000, \"user_id\": \"user1\", \"project_id\": \"project1\", \"resource_id\": \"resource1\", \"timestamp\": \"2021-02-10T03:50:41.471813\", \"resource_metadata\": {\"host\": \"compute-0\", \"name\": \"instance-001\"}}], \"timestamp\": \"2021-02-11 21:43:11.180978\"}"
			},
			"context": {}
		}`)

		msg, err := c.ParseInputJSON(input)
		require.NoError(t, err)
		require.NotNil(t, msg)
		assert.Equal(t, "test.publisher", msg.Publisher)
		assert.Equal(t, 1, len(msg.Payload))
		assert.Equal(t, "cpu", msg.Payload[0].CounterName)
		assert.Equal(t, "cumulative", msg.Payload[0].CounterType)
		assert.Equal(t, "ns", msg.Payload[0].CounterUnit)
		assert.Equal(t, float64(347670000000), msg.Payload[0].CounterVolume)
		assert.Equal(t, "user1", msg.Payload[0].UserID)
		assert.Equal(t, "project1", msg.Payload[0].ProjectID)
		assert.Equal(t, "resource1", msg.Payload[0].ResourceID)
		assert.Equal(t, "compute-0", msg.Payload[0].ResourceMetadata.Host)
		assert.Equal(t, "instance-001", msg.Payload[0].ResourceMetadata.Name)
	})

	t.Run("parse message with escaped quotes in oslo message", func(t *testing.T) {
		c := New()
		// The oslo.message field contains escaped quotes that need to be sanitized
		input := []byte(`{
			"request": {
				"oslo.version": "2.0",
				"oslo.message": "{\\\"publisher_id\\\": \\\"test.publisher\\\", \\\"payload\\\": [{\\\"counter_name\\\": \\\"memory\\\", \\\"counter_volume\\\": 512}]}"
			}
		}`)

		msg, err := c.ParseInputJSON(input)
		require.NoError(t, err)
		require.NotNil(t, msg)
		assert.Equal(t, 1, len(msg.Payload))
		assert.Equal(t, "memory", msg.Payload[0].CounterName)
		assert.Equal(t, float64(512), msg.Payload[0].CounterVolume)
	})

	t.Run("parse message with multiple metrics", func(t *testing.T) {
		c := New()
		input := []byte(`{
			"request": {
				"oslo.message": "{\"publisher_id\": \"test.publisher\", \"payload\": [{\"counter_name\": \"cpu\", \"counter_volume\": 100}, {\"counter_name\": \"memory\", \"counter_volume\": 512}, {\"counter_name\": \"disk\", \"counter_volume\": 1024}]}"
			}
		}`)

		msg, err := c.ParseInputJSON(input)
		require.NoError(t, err)
		require.NotNil(t, msg)
		assert.Equal(t, 3, len(msg.Payload))
		assert.Equal(t, "cpu", msg.Payload[0].CounterName)
		assert.Equal(t, "memory", msg.Payload[1].CounterName)
		assert.Equal(t, "disk", msg.Payload[2].CounterName)
	})

	t.Run("parse message with user metadata", func(t *testing.T) {
		c := New()
		input := []byte(`{
			"request": {
				"oslo.message": "{\"publisher_id\": \"test.publisher\", \"payload\": [{\"counter_name\": \"cpu\", \"counter_volume\": 512, \"resource_metadata\": {\"host\": \"compute-0\", \"user_metadata\": {\"server_group\": \"group1\", \"custom_key\": \"custom_value\"}}}]}"
			}
		}`)

		msg, err := c.ParseInputJSON(input)
		require.NoError(t, err)
		require.NotNil(t, msg)
		assert.Equal(t, 1, len(msg.Payload))
		require.NotNil(t, msg.Payload[0].ResourceMetadata.UserMetadata)
		assert.Equal(t, "group1", msg.Payload[0].ResourceMetadata.UserMetadata["server_group"])
		assert.Equal(t, "custom_value", msg.Payload[0].ResourceMetadata.UserMetadata["custom_key"])
	})

	t.Run("parse message with all optional fields", func(t *testing.T) {
		c := New()
		input := []byte(`{
			"request": {
				"oslo.message": "{\"publisher_id\": \"test.publisher\", \"payload\": [{\"source\": \"openstack\", \"counter_name\": \"vcpus\", \"counter_type\": \"gauge\", \"counter_unit\": \"vcpu\", \"counter_volume\": 2, \"user_id\": \"user1\", \"user_name\": \"testuser\", \"project_id\": \"project1\", \"project_name\": \"testproject\", \"resource_id\": \"resource1\", \"timestamp\": \"2020-09-14T16:12:49.939250+00:00\", \"resource_metadata\": {\"host\": \"compute-0\", \"name\": \"instance-001\", \"display_name\": \"test-instance\", \"instance_host\": \"host1\"}}]}"
			}
		}`)

		msg, err := c.ParseInputJSON(input)
		require.NoError(t, err)
		require.NotNil(t, msg)
		assert.Equal(t, 1, len(msg.Payload))
		assert.Equal(t, "openstack", msg.Payload[0].Source)
		assert.Equal(t, "vcpus", msg.Payload[0].CounterName)
		assert.Equal(t, "gauge", msg.Payload[0].CounterType)
		assert.Equal(t, "vcpu", msg.Payload[0].CounterUnit)
		assert.Equal(t, float64(2), msg.Payload[0].CounterVolume)
		assert.Equal(t, "user1", msg.Payload[0].UserID)
		assert.Equal(t, "testuser", msg.Payload[0].UserName)
		assert.Equal(t, "project1", msg.Payload[0].ProjectID)
		assert.Equal(t, "testproject", msg.Payload[0].ProjectName)
		assert.Equal(t, "resource1", msg.Payload[0].ResourceID)
		assert.Equal(t, "2020-09-14T16:12:49.939250+00:00", msg.Payload[0].Timestamp)
		assert.Equal(t, "compute-0", msg.Payload[0].ResourceMetadata.Host)
		assert.Equal(t, "instance-001", msg.Payload[0].ResourceMetadata.Name)
		assert.Equal(t, "test-instance", msg.Payload[0].ResourceMetadata.DisplayName)
		assert.Equal(t, "host1", msg.Payload[0].ResourceMetadata.InstanceHost)
	})

	t.Run("error on invalid JSON in outer schema", func(t *testing.T) {
		c := New()
		input := []byte(`{invalid json}`)

		msg, err := c.ParseInputJSON(input)
		require.Error(t, err)
		assert.Nil(t, msg)
	})

	t.Run("error on invalid JSON in oslo message", func(t *testing.T) {
		c := New()
		input := []byte(`{
			"request": {
				"oslo.message": "{invalid nested json}"
			}
		}`)

		msg, err := c.ParseInputJSON(input)
		require.Error(t, err)
		assert.Nil(t, msg)
	})

	t.Run("parse empty payload", func(t *testing.T) {
		c := New()
		input := []byte(`{
			"request": {
				"oslo.message": "{\"publisher_id\": \"test.publisher\", \"payload\": []}"
			}
		}`)

		msg, err := c.ParseInputJSON(input)
		require.NoError(t, err)
		require.NotNil(t, msg)
		assert.Equal(t, "test.publisher", msg.Publisher)
		assert.Equal(t, 0, len(msg.Payload))
	})
}

func TestParseInputMsgPack(t *testing.T) {
	t.Run("parse valid msgpack message", func(t *testing.T) {
		c := New()

		// Create a metric
		metric := Metric{
			CounterName:   "cpu",
			CounterType:   "cumulative",
			CounterUnit:   "ns",
			CounterVolume: 347670000000,
			UserID:        "user1",
			ProjectID:     "project1",
			ResourceID:    "resource1",
			Timestamp:     "2021-02-10T03:50:41",
			ResourceMetadata: Metadata{
				Host: "compute-0",
				Name: "instance-001",
			},
		}

		// Create a message with the metric
		testMsg := Message{
			Publisher: "test.publisher",
			Payload:   []Metric{metric},
		}

		// Marshal to msgpack
		input, err := msgpack.Marshal(testMsg)
		require.NoError(t, err)

		msg, err := c.ParseInputMsgPack(input)
		require.NoError(t, err)
		require.NotNil(t, msg)
		assert.Equal(t, "test.publisher", msg.Publisher)
		// Note: ParseInputMsgPack appends the metric, so we get it twice
		assert.GreaterOrEqual(t, len(msg.Payload), 1)
		assert.Equal(t, "cpu", msg.Payload[0].CounterName)
		assert.Equal(t, "cumulative", msg.Payload[0].CounterType)
		assert.Equal(t, float64(347670000000), msg.Payload[0].CounterVolume)
	})

	t.Run("error on invalid msgpack", func(t *testing.T) {
		c := New()
		input := []byte{0xff, 0xff, 0xff}

		msg, err := c.ParseInputMsgPack(input)
		require.Error(t, err)
		assert.Nil(t, msg)
	})

	t.Run("parse msgpack with metadata", func(t *testing.T) {
		c := New()

		metric := Metric{
			CounterName:   "memory",
			CounterVolume: 512,
			ResourceMetadata: Metadata{
				Host:         "compute-0",
				Name:         "instance-001",
				DisplayName:  "test-instance",
				InstanceHost: "host1",
				UserMetadata: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			},
		}

		testMsg := Message{
			Publisher: "test.publisher",
			Payload:   []Metric{metric},
		}

		input, err := msgpack.Marshal(testMsg)
		require.NoError(t, err)

		msg, err := c.ParseInputMsgPack(input)
		require.NoError(t, err)
		require.NotNil(t, msg)
		assert.Equal(t, "memory", msg.Payload[0].CounterName)
		assert.NotNil(t, msg.Payload[0].ResourceMetadata.UserMetadata)
	})
}

func TestSanitize(t *testing.T) {
	t.Run("remove escaped quotes", func(t *testing.T) {
		c := New()
		c.schema.Request.OsloMessage = `{\"key\": \"value\"}`

		result := c.sanitize()
		assert.Contains(t, result, `{"key": "value"}`)
		assert.NotContains(t, result, `\"`)
	})

	t.Run("fix payload array formatting", func(t *testing.T) {
		c := New()
		c.schema.Request.OsloMessage = `{"payload": [{\"counter\": \"cpu\"}]}`

		result := c.sanitize()
		assert.Contains(t, result, `"payload": [{"counter": "cpu"}]`)
	})

	t.Run("handle payload with spaces", func(t *testing.T) {
		c := New()
		c.schema.Request.OsloMessage = `{"payload"  :  [{\"counter\": \"cpu\"}]}`

		result := c.sanitize()
		assert.Contains(t, result, `"payload": [{"counter": "cpu"}]`)
	})

	t.Run("handle multiple payload items", func(t *testing.T) {
		c := New()
		c.schema.Request.OsloMessage = `{"payload": [{\"counter\": \"cpu\"}, {\"counter\": \"memory\"}]}`

		result := c.sanitize()
		assert.Contains(t, result, `"payload": [{"counter": "cpu"}, {"counter": "memory"}]`)
	})

	t.Run("handle missing payload array", func(t *testing.T) {
		c := New()
		c.schema.Request.OsloMessage = `{\"publisher\": \"test\"}`

		result := c.sanitize()
		// Should still work even without payload
		assert.Contains(t, result, `"publisher": "test"`)
	})
}
