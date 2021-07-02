package main

import (
	"context"
	"fmt"
	"time"

	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/handler"
	"github.com/infrawatch/sg-core/plugins/handler/events/pkg/lib"
	"github.com/infrawatch/sg-core/plugins/handler/sensubility-metrics/pkg/sensu"
	jsoniter "github.com/json-iterator/go"
)

var (
	json       = jsoniter.ConfigCompatibleWithStandardLibrary
	metricName = "sensubility_container_health_status"
)

type sensubilityMetrics struct {
	totalMetricsDecoded   int64
	totalDecodeErrors     int64
	totalMessagesReceived int64
}

func (sm *sensubilityMetrics) Run(ctx context.Context, mpf bus.MetricPublishFunc, epf bus.EventPublishFunc) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
			mpf(
				"sg_total_sensubility_metric_decode_count",
				0,
				data.COUNTER,
				0,
				float64(sm.totalMetricsDecoded),
				[]string{"source"},
				[]string{"SG"},
			)
			mpf(
				"sg_total_sensubility_metric_decode_error_count",
				0,
				data.COUNTER,
				0,
				float64(sm.totalDecodeErrors),
				[]string{"source"},
				[]string{"SG"},
			)
			mpf(
				"sg_total_sensubility_msg_received_count",
				0,
				data.COUNTER,
				0,
				float64(sm.totalMessagesReceived),
				[]string{"source"},
				[]string{"SG"},
			)
		}
	}
}

func (sm *sensubilityMetrics) Handle(blob []byte, reportErrors bool, mpf bus.MetricPublishFunc, epf bus.EventPublishFunc) error {
	sm.totalMessagesReceived++
	sensuMsg := sensu.Message{}
	err := json.Unmarshal(blob, &sensuMsg)
	if err != nil {
		return err
	}

	if !sensu.IsMsgValid(sensuMsg) {
		sm.totalDecodeErrors++
		err := sensu.BuildMsgErr(sensuMsg)
		if reportErrors {
			sm.publishErrEvent(err, epf)
		}
		return err
	}

	outputs := sensu.HealthCheckOutput{}
	err = json.Unmarshal([]byte(sensuMsg.Annotations.Output), &outputs)
	if err != nil {
		return err
	}

	if !sensu.IsOutputValid(outputs) {
		sm.totalDecodeErrors++
		err := sensu.BuildOutputsErr(outputs)
		if reportErrors {
			sm.publishErrEvent(err, epf)
		}
		sm.totalDecodeErrors += int64(len(err.(*sensu.ErrMissingFields).Fields))
		return err
	}

	epoc := lib.EpochFromFormat(sensuMsg.StartsAt)
	if epoc == 0 {
		return fmt.Errorf("failed determining epoch time from timestamp '%s'", sensuMsg.StartsAt)
	}

	for _, output := range outputs {
		sm.totalMetricsDecoded++
		mpf(
			metricName,
			float64(epoc),
			data.GAUGE,
			time.Second*10, // TODO: figure out what a good interval is, or make configure-able
			output.Healthy,
			[]string{"container", "host"},
			[]string{output.Service, sensuMsg.Labels.Client},
		)
	}
	return nil
}

func (sm *sensubilityMetrics) Identify() string {
	return "sensubility-metrics"
}

func (sm *sensubilityMetrics) Config(blob []byte) error {
	return nil
}

func New() handler.Handler {
	return &sensubilityMetrics{}
}

func (sm *sensubilityMetrics) publishErrEvent(err error, epf bus.EventPublishFunc) {
	epf(data.Event{
		Index:    sm.Identify(),
		Type:     data.ERROR,
		Severity: data.CRITICAL,
		Time:     0.0,
		Labels: map[string]interface{}{
			"error":   err.Error(),
			"message": "failed to parse event - disregarding",
		},
		Annotations: map[string]interface{}{
			"description": "internal smartgateway sensubility handler error",
		},
	})
}
