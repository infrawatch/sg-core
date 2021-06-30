package main

import (
	"testing"
	"time"

	"github.com/infrawatch/sg-core/plugins/handler/sensubility-metrics/pkg/sensu"

	"github.com/infrawatch/sg-core/pkg/data"
	"gopkg.in/go-playground/assert.v1"
)

// for convenience so that the whole metric publisher function signature
// does not need to be written out for every test
type mpFuncWrapper struct {
	mFunc func(data.Metric)
}

func (pf *mpFuncWrapper) MPFunc(name string, timestamp float64, mt data.MetricType, interval time.Duration, val float64, labelKeys []string, labelVals []string) {
	pf.mFunc(data.Metric{
		Name:      name,
		Time:      timestamp,
		Type:      mt,
		Interval:  interval,
		Value:     val,
		LabelKeys: labelKeys,
		LabelVals: labelVals,
	})
}

func nilEPFunc(data.Event) {

}

func TestSensuMetricHandling(t *testing.T) {
	plug := sensubilityMetrics{}

	healthCheckRes := sensu.HealthCheckOutput{
		{
			Service:   "glance",
			Container: 1235,
			Status:    "healthy",
			Healthy:   1,
		},
		{
			Service:   "nova",
			Container: 1235,
			Status:    "healthy",
			Healthy:   0,
		},
	}

	healthCheckResBlob, err := json.Marshal(healthCheckRes)
	if err != nil {
		t.Error(err)
	}

	input := sensu.Message{
		Labels: sensu.Labels{
			Client:   "controller-0.osp-cloudops-0",
			Check:    "check-container-health",
			Severity: "FAILURE",
		},
		Annotations: sensu.Annotations{
			Output: string(healthCheckResBlob),
		},
		StartsAt: "2021-06-29T18:49:13Z",
	}

	t.Run("correct metric generation", func(t *testing.T) {
		correctResults := []data.Metric{
			{
				Name:      "check_container_health",
				Time:      1625006953.0,
				Type:      data.GAUGE,
				Interval:  10,
				Value:     1,
				LabelKeys: []string{"service", "host"},
				LabelVals: []string{"glance", "controller-0.osp-cloudops-0"},
			},
			{
				Name:      "check_container_health",
				Time:      1625006953.0,
				Type:      data.GAUGE,
				Interval:  10,
				Value:     0,
				LabelKeys: []string{"service", "host"},
				LabelVals: []string{"nova", "controller-0.osp-cloudops-0"},
			},
		}
		numPubCalls := 0
		pubWrapper := mpFuncWrapper{
			mFunc: func(m data.Metric) {
				assert.Equal(t,
					m,
					correctResults[numPubCalls],
				)
				numPubCalls++
			},
		}
		inputBlob, err := json.Marshal(input)
		if err != nil {
			t.Error(err)
		}

		err = plug.Handle(
			inputBlob,
			false,
			pubWrapper.MPFunc,
			nilEPFunc,
		)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("metric name generation", func(t *testing.T) {
		t.Run("missing name elements", func(t *testing.T) {
			input.Labels.Check = ""
			blob, err := json.Marshal(input)
			if err != nil {
				t.Error(err)
			}

			pubWrapper := mpFuncWrapper{
				mFunc: func(data.Metric) {
					t.Error("publish func should not have been called")
				},
			}
			err = plug.Handle(
				blob,
				false,
				pubWrapper.MPFunc,
				nilEPFunc,
			)
			assert.NotEqual(t, err, nil)
		})

		t.Run("from check field", func(t *testing.T) {
			input.Labels.Check = "check"

			blob, err := json.Marshal(input)
			if err != nil {
				t.Error(err)
			}

			pubWrapper := mpFuncWrapper{
				mFunc: func(m data.Metric) {
					assert.Equal(t, "check", m.Name)
				},
			}

			err = plug.Handle(
				blob,
				false,
				pubWrapper.MPFunc,
				nilEPFunc,
			)
			if err != nil {
				t.Error(err)
			}
		})

		t.Run("sanitized name", func(t *testing.T) {
			input.Labels.Check = "check.container?#$!-health"

			pubWrapper := mpFuncWrapper{
				mFunc: func(m data.Metric) {
					assert.Equal(t, "check_container_health", m.Name)
				},
			}

			blob, err := json.Marshal(input)
			if err != nil {
				t.Error(err)
			}

			err = plug.Handle(
				blob,
				false,
				pubWrapper.MPFunc,
				nilEPFunc,
			)
			if err != nil {
				t.Error(err)
			}
		})
	})

	t.Run("error handling", func(t *testing.T) {
		t.Run("corrupted JSON", func(t *testing.T) {
			plug := sensubilityMetrics{}
			pubWrapper := mpFuncWrapper{
				mFunc: func(data.Metric) {
					t.Error("publish func should not have been called")
				},
			}

			err := plug.Handle(
				[]byte("{"),
				false,
				pubWrapper.MPFunc,
				nilEPFunc,
			)
			assert.NotEqual(t, err, nil)
		})

		t.Run("missing field err generation", func(t *testing.T) {
			plug := sensubilityMetrics{}
			pubWrapper := mpFuncWrapper{
				mFunc: func(data.Metric) {
					t.Error("publish func should not have been called")
				},
			}

			err := plug.Handle(
				[]byte("{}"),
				false,
				pubWrapper.MPFunc,
				nilEPFunc,
			)
			eE, ok := err.(*ErrMissingFields)
			assert.Equal(t, ok, true)
			assert.Equal(t, eE.fields, []string{
				"startsAt",
				"labels.check",
				"client",
			})
		})
	})
	t.Run("time", func(t *testing.T) {
		t.Run("invalid time", func(t *testing.T) {
			plug := sensubilityMetrics{}
			pubWrapper := mpFuncWrapper{
				mFunc: func(data.Metric) {
					t.Error("publish func should not have been called")
				},
			}
			input.StartsAt = "asd4"
			blob, err := json.Marshal(input)
			if err != nil {
				t.Error(err)
			}

			err = plug.Handle(
				blob,
				false,
				pubWrapper.MPFunc,
				nilEPFunc,
			)
			assert.NotEqual(t, err, nil)
		})
	})
	t.Run("output field", func(t *testing.T) {
		t.Run("incorrect format", func(t *testing.T) {
			plug := sensubilityMetrics{}
			pubWrapper := mpFuncWrapper{
				mFunc: func(data.Metric) {
					t.Error("publish func should not have been called")
				},
			}
			input.Annotations.Output = "asd4"
			blob, err := json.Marshal(input)
			if err != nil {
				t.Error(err)
			}

			err = plug.Handle(
				blob,
				false,
				pubWrapper.MPFunc,
				nilEPFunc,
			)
			assert.NotEqual(t, err, nil)
		})
		t.Run("missing fields", func(t *testing.T) {
			plug := sensubilityMetrics{}
			pubWrapper := mpFuncWrapper{
				mFunc: func(data.Metric) {
					t.Error("publish func should not have been called")
				},
			}

			input.Annotations.Output = "[{}]"
			blob, err := json.Marshal(input)
			if err != nil {
				t.Error(err)
			}

			err = plug.Handle(
				blob,
				false,
				pubWrapper.MPFunc,
				nilEPFunc,
			)
			eE, ok := err.(*ErrMissingFields)
			assert.Equal(t, ok, true)
			assert.Equal(t, eE.fields, []string{
				"annotations.output.service",
				"annotations.output.healthy",
			})
		})
	})
}

// test incorrect format in output field
// test time field generation

// Name      string
// Time      float64
// Type      MetricType
// Interval  time.Duration
// Value     float64
// LabelKeys []string
// LabelVals []string

// Name:  labels.check (replace '-' with '_'), removing
// Time: StartsAt
// Type: data.GAUGE
// Interval: ??
// Value: annotations.status
// LabelKeys: []string{'service', 'host'}
// LabelVals: []string{'annotations.output.service','client'}

// correct parsing
