package sensu

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsMsgValid(t *testing.T) {
	t.Run("valid message with all required fields", func(t *testing.T) {
		msg := Message{
			StartsAt: "2023-01-01T00:00:00Z",
			Labels: Labels{
				Client:   "test-client",
				Check:    "test-check",
				Severity: "warning",
			},
			Annotations: Annotations{
				Command: "test-command",
				Output:  "test output",
			},
		}

		assert.True(t, IsMsgValid(msg))
	})

	t.Run("valid message with minimal required fields", func(t *testing.T) {
		msg := Message{
			StartsAt: "2023-01-01T00:00:00Z",
			Labels: Labels{
				Client: "test-client",
			},
		}

		assert.True(t, IsMsgValid(msg))
	})

	t.Run("invalid message with missing StartsAt", func(t *testing.T) {
		msg := Message{
			Labels: Labels{
				Client: "test-client",
			},
		}

		assert.False(t, IsMsgValid(msg))
	})

	t.Run("invalid message with empty StartsAt", func(t *testing.T) {
		msg := Message{
			StartsAt: "",
			Labels: Labels{
				Client: "test-client",
			},
		}

		assert.False(t, IsMsgValid(msg))
	})

	t.Run("invalid message with missing Client", func(t *testing.T) {
		msg := Message{
			StartsAt: "2023-01-01T00:00:00Z",
			Labels:   Labels{},
		}

		assert.False(t, IsMsgValid(msg))
	})

	t.Run("invalid message with empty Client", func(t *testing.T) {
		msg := Message{
			StartsAt: "2023-01-01T00:00:00Z",
			Labels: Labels{
				Client: "",
			},
		}

		assert.False(t, IsMsgValid(msg))
	})

	t.Run("invalid message with both fields missing", func(t *testing.T) {
		msg := Message{}

		assert.False(t, IsMsgValid(msg))
	})

	t.Run("valid message with optional fields", func(t *testing.T) {
		msg := Message{
			StartsAt: "2023-01-01T00:00:00Z",
			Labels: Labels{
				Client:   "test-client",
				Check:    "disk-usage",
				Severity: "critical",
			},
			Annotations: Annotations{
				Command:  "/usr/bin/check_disk",
				Issued:   1234567890,
				Executed: 1234567891,
				Duration: 1.5,
				Output:   "CRITICAL - disk usage at 95%",
				Status:   2,
				StartsAt: "2023-01-01T00:00:00Z",
			},
		}

		assert.True(t, IsMsgValid(msg))
	})
}

func TestIsOutputValid(t *testing.T) {
	t.Run("valid output with single service", func(t *testing.T) {
		outputs := HealthCheckOutput{
			{
				Service:   "test-service",
				Container: "test-container",
				Status:    "running",
				Healthy:   1.0,
			},
		}

		assert.True(t, IsOutputValid(outputs))
	})

	t.Run("valid output with multiple services", func(t *testing.T) {
		outputs := HealthCheckOutput{
			{
				Service:   "service1",
				Container: "container1",
				Status:    "running",
				Healthy:   1.0,
			},
			{
				Service:   "service2",
				Container: "container2",
				Status:    "stopped",
				Healthy:   0.0,
			},
			{
				Service:   "service3",
				Container: "container3",
				Status:    "running",
				Healthy:   1.0,
			},
		}

		assert.True(t, IsOutputValid(outputs))
	})

	t.Run("valid output with minimal fields", func(t *testing.T) {
		outputs := HealthCheckOutput{
			{
				Service: "minimal-service",
			},
		}

		assert.True(t, IsOutputValid(outputs))
	})

	t.Run("invalid output with missing service", func(t *testing.T) {
		outputs := HealthCheckOutput{
			{
				Container: "test-container",
				Status:    "running",
				Healthy:   1.0,
			},
		}

		assert.False(t, IsOutputValid(outputs))
	})

	t.Run("invalid output with empty service", func(t *testing.T) {
		outputs := HealthCheckOutput{
			{
				Service:   "",
				Container: "test-container",
			},
		}

		assert.False(t, IsOutputValid(outputs))
	})

	t.Run("invalid output with one valid and one invalid", func(t *testing.T) {
		outputs := HealthCheckOutput{
			{
				Service:   "valid-service",
				Container: "container1",
			},
			{
				Service:   "",
				Container: "container2",
			},
		}

		assert.False(t, IsOutputValid(outputs))
	})

	t.Run("valid empty output array", func(t *testing.T) {
		outputs := HealthCheckOutput{}

		assert.True(t, IsOutputValid(outputs))
	})

	t.Run("invalid output with missing service in middle", func(t *testing.T) {
		outputs := HealthCheckOutput{
			{
				Service: "service1",
			},
			{
				Service: "",
			},
			{
				Service: "service3",
			},
		}

		assert.False(t, IsOutputValid(outputs))
	})
}

func TestBuildMsgErr(t *testing.T) {
	t.Run("error with missing StartsAt", func(t *testing.T) {
		msg := Message{
			Labels: Labels{
				Client: "test-client",
			},
		}

		err := BuildMsgErr(msg)
		require.NotNil(t, err)

		eMF, ok := err.(*ErrMissingFields)
		require.True(t, ok)
		assert.Contains(t, eMF.Fields, "startsAt")
		assert.NotContains(t, eMF.Fields, "labels.client")
		assert.Contains(t, err.Error(), "startsAt")
	})

	t.Run("error with missing Client", func(t *testing.T) {
		msg := Message{
			StartsAt: "2023-01-01T00:00:00Z",
		}

		err := BuildMsgErr(msg)
		require.NotNil(t, err)

		eMF, ok := err.(*ErrMissingFields)
		require.True(t, ok)
		assert.Contains(t, eMF.Fields, "labels.client")
		assert.NotContains(t, eMF.Fields, "startsAt")
		assert.Contains(t, err.Error(), "labels.client")
	})

	t.Run("error with both fields missing", func(t *testing.T) {
		msg := Message{}

		err := BuildMsgErr(msg)
		require.NotNil(t, err)

		eMF, ok := err.(*ErrMissingFields)
		require.True(t, ok)
		assert.Contains(t, eMF.Fields, "startsAt")
		assert.Contains(t, eMF.Fields, "labels.client")
		assert.Contains(t, err.Error(), "startsAt")
		assert.Contains(t, err.Error(), "labels.client")
		assert.Contains(t, err.Error(), "missing fields in received data")
	})

	t.Run("error with valid message returns empty error", func(t *testing.T) {
		msg := Message{
			StartsAt: "2023-01-01T00:00:00Z",
			Labels: Labels{
				Client: "test-client",
			},
		}

		err := BuildMsgErr(msg)
		require.NotNil(t, err)

		eMF, ok := err.(*ErrMissingFields)
		require.True(t, ok)
		assert.Empty(t, eMF.Fields)
	})

	t.Run("error message format", func(t *testing.T) {
		msg := Message{}

		err := BuildMsgErr(msg)
		require.NotNil(t, err)

		errorMsg := err.Error()
		assert.Contains(t, errorMsg, "missing fields in received data")
		assert.Contains(t, errorMsg, "(")
		assert.Contains(t, errorMsg, ")")
	})
}

func TestBuildOutputsErr(t *testing.T) {
	t.Run("error with single missing service", func(t *testing.T) {
		outputs := HealthCheckOutput{
			{
				Container: "test-container",
			},
		}

		err := BuildOutputsErr(outputs)
		require.NotNil(t, err)

		eMF, ok := err.(*ErrMissingFields)
		require.True(t, ok)
		assert.Contains(t, eMF.Fields, "annotations.output[0].service")
		assert.Contains(t, err.Error(), "annotations.output[0].service")
	})

	t.Run("error with multiple missing services", func(t *testing.T) {
		outputs := HealthCheckOutput{
			{
				Container: "container1",
			},
			{
				Service: "valid-service",
			},
			{
				Container: "container3",
			},
		}

		err := BuildOutputsErr(outputs)
		require.NotNil(t, err)

		eMF, ok := err.(*ErrMissingFields)
		require.True(t, ok)
		assert.Contains(t, eMF.Fields, "annotations.output[0].service")
		assert.Contains(t, eMF.Fields, "annotations.output[2].service")
		assert.NotContains(t, eMF.Fields, "annotations.output[1].service")
	})

	t.Run("error with all services missing", func(t *testing.T) {
		outputs := HealthCheckOutput{
			{
				Container: "container1",
			},
			{
				Container: "container2",
			},
			{
				Container: "container3",
			},
		}

		err := BuildOutputsErr(outputs)
		require.NotNil(t, err)

		eMF, ok := err.(*ErrMissingFields)
		require.True(t, ok)
		assert.Len(t, eMF.Fields, 3)
		assert.Contains(t, eMF.Fields, "annotations.output[0].service")
		assert.Contains(t, eMF.Fields, "annotations.output[1].service")
		assert.Contains(t, eMF.Fields, "annotations.output[2].service")
	})

	t.Run("error with valid outputs returns empty error", func(t *testing.T) {
		outputs := HealthCheckOutput{
			{
				Service:   "service1",
				Container: "container1",
			},
			{
				Service:   "service2",
				Container: "container2",
			},
		}

		err := BuildOutputsErr(outputs)
		require.NotNil(t, err)

		eMF, ok := err.(*ErrMissingFields)
		require.True(t, ok)
		assert.Empty(t, eMF.Fields)
	})

	t.Run("error with empty outputs array", func(t *testing.T) {
		outputs := HealthCheckOutput{}

		err := BuildOutputsErr(outputs)
		require.NotNil(t, err)

		eMF, ok := err.(*ErrMissingFields)
		require.True(t, ok)
		assert.Empty(t, eMF.Fields)
	})

	t.Run("error index format in message", func(t *testing.T) {
		outputs := HealthCheckOutput{
			{},
			{},
			{},
		}

		err := BuildOutputsErr(outputs)
		require.NotNil(t, err)

		errorMsg := err.Error()
		assert.Contains(t, errorMsg, "[0]")
		assert.Contains(t, errorMsg, "[1]")
		assert.Contains(t, errorMsg, "[2]")
	})
}

func TestErrMissingFields(t *testing.T) {
	t.Run("error message with single field", func(t *testing.T) {
		err := &ErrMissingFields{
			Fields: []string{"field1"},
		}

		assert.Equal(t, "missing fields in received data (field1)", err.Error())
	})

	t.Run("error message with multiple fields", func(t *testing.T) {
		err := &ErrMissingFields{
			Fields: []string{"field1", "field2", "field3"},
		}

		errorMsg := err.Error()
		assert.Contains(t, errorMsg, "missing fields in received data")
		assert.Contains(t, errorMsg, "field1")
		assert.Contains(t, errorMsg, "field2")
		assert.Contains(t, errorMsg, "field3")
		assert.Contains(t, errorMsg, ", ")
	})

	t.Run("error message with empty fields", func(t *testing.T) {
		err := &ErrMissingFields{
			Fields: []string{},
		}

		assert.Equal(t, "missing fields in received data ()", err.Error())
	})

	t.Run("add missing field", func(t *testing.T) {
		err := &ErrMissingFields{
			Fields: []string{},
		}

		err.addMissingField("field1")
		assert.Contains(t, err.Fields, "field1")
		assert.Len(t, err.Fields, 1)

		err.addMissingField("field2")
		assert.Contains(t, err.Fields, "field1")
		assert.Contains(t, err.Fields, "field2")
		assert.Len(t, err.Fields, 2)
	})

	t.Run("add multiple missing fields", func(t *testing.T) {
		err := &ErrMissingFields{}

		err.addMissingField("field1")
		err.addMissingField("field2")
		err.addMissingField("field3")

		assert.Len(t, err.Fields, 3)
		assert.Equal(t, "field1", err.Fields[0])
		assert.Equal(t, "field2", err.Fields[1])
		assert.Equal(t, "field3", err.Fields[2])
	})

	t.Run("error message format with long field names", func(t *testing.T) {
		err := &ErrMissingFields{
			Fields: []string{
				"annotations.output[0].service",
				"annotations.output[1].service",
				"labels.client",
			},
		}

		errorMsg := err.Error()
		assert.Contains(t, errorMsg, "annotations.output[0].service")
		assert.Contains(t, errorMsg, "annotations.output[1].service")
		assert.Contains(t, errorMsg, "labels.client")
	})
}
