package main

import (
	"fmt"
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

func TestConfiguration(t *testing.T) {
	plug := sensubilityMetrics{}

	t.Run("default config", func(t *testing.T) {
		err := plug.Config([]byte(""))
		if err != nil {
			t.Error(err)
		}

		if plug.configuration.MetricInterval != 10 {
			t.Errorf("default metricInterval should be 10, got %d", plug.configuration.MetricInterval)
		}
	})

	t.Run("adjusted", func(t *testing.T) {
		configuration := "metricInterval: 50"

		err := plug.Config([]byte(configuration))
		if err != nil {
			t.Error(err)
		}

		if plug.configuration.MetricInterval != 50 {
			t.Errorf("loading configuration failed - expected metricInterval: 50, got %d", plug.configuration.MetricInterval)
		}
	})
}

func TestSensuMetricHandling(t *testing.T) {
	plug := sensubilityMetrics{}

	healthCheckRes := sensu.HealthCheckOutput{
		{
			Service:   "glance",
			Container: "1235",
			Status:    "healthy",
			Healthy:   1,
		},
		{
			Service:   "nova",
			Container: "1235",
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

	t.Run("full metric generation", func(t *testing.T) {
		correctResults := []data.Metric{
			{
				Name:      "container_health_status",
				Time:      1624992553.0,
				Type:      data.GAUGE,
				Interval:  time.Second * 10,
				Value:     1,
				LabelKeys: []string{"service", "host"},
				LabelVals: []string{"glance", "controller-0.osp-cloudops-0"},
			},
			{
				Name:      "container_health_status",
				Time:      1624992553.0,
				Type:      data.GAUGE,
				Interval:  time.Second * 10,
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

		fmt.Println(string(inputBlob))
		err = plug.Handle(
			inputBlob,
			false,
			pubWrapper.MPFunc,
			nilEPFunc,
		)
		if err != nil {
			t.Error(err)
		}
		if !(numPubCalls > 0) {
			t.Error("publish function never called")
		}
	})

	t.Run("metric name data model", func(t *testing.T) {
		pubWrapper := mpFuncWrapper{
			mFunc: func(m data.Metric) {
				assert.Equal(t, "container_health_status", m.Name)
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
			eE, ok := err.(*sensu.ErrMissingFields)
			assert.Equal(t, ok, true)
			assert.Equal(t, eE.Fields, []string{
				"startsAt",
				"labels.client",
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

			input.Annotations.Output = "[]"
			input.StartsAt = "2021-06-29T18:49:13Z"
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

			input.Annotations.Output = "[{},{}]"
			blob, err = json.Marshal(input)
			if err != nil {
				t.Error(err)
			}
			err = plug.Handle(
				blob,
				false,
				pubWrapper.MPFunc,
				nilEPFunc,
			)
			eE, ok := err.(*sensu.ErrMissingFields)
			assert.Equal(t, ok, true)
			assert.Equal(t, eE.Fields, []string{
				"annotations.output[0].service",
				"annotations.output[1].service",
			})
		})
	})
}
