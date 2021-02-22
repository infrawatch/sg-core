package main

import (
	"bytes"
	"context"
	"encoding/json"
	"math/rand"
	"time"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/transport"
)

type collectdMetric struct {
	Values         []float64 `json:"values"`
	Dstypes        []string  `json:"dstypes"`
	Dsnames        []string  `json:"dsnames"`
	Time           int64     `json:"time"`
	Interval       int       `json:"interval"`
	Host           string    `json:"host"`
	Plugin         string    `json:"plugin"`
	PluginInstance string    `json:"plugin_instance"`
	Type           string    `json:"type"`
	TypeInstance   string    `json:"type_instance"`
}

var ceilometerMessage = `{"request": {"oslo.version": "2.0", "oslo.message": "{\"message_id\": \"b6f21ada-8465-49b8-9af2-fbf4c0ec1dbe\", \"publisher_id\": \"telemetry.publisher.controller-0.redhat.local\", \"event_type\": \"metering\", \"priority\": \"SAMPLE\", \"payload\": [{\"source\": \"openstack\", \"counter_name\": \"cpu\", \"counter_type\": \"cumulative\", \"counter_unit\": \"ns\", \"counter_volume\": 347670000000, \"user_id\": \"581d4d733fad4baebd0edecb5dc6b889\", \"project_id\": \"40390761eaf7414c8125917efc21024c\", \"resource_id\": \"db4aee71-6c98-4505-bdf2-36d987e32be7\", \"timestamp\": \"2021-02-10T03:50:41.471813\", \"resource_metadata\": {\"display_name\": \"RED\", \"name\": \"instance-00000004\", \"instance_id\": \"db4aee71-6c98-4505-bdf2-36d987e32be7\", \"instance_type\": \"m1.tiny\", \"host\": \"fad58495f1d0d88347ebef0a41f13c81081f1454ac3ff61c711144d5\", \"instance_host\": \"compute-0.redhat.local\", \"flavor\": {\"id\": \"beeaf8e5-2bda-4b3c-88ad-cf41ad4df149\", \"name\": \"m1.tiny\", \"vcpus\": 2, \"ram\": 512, \"disk\": 1, \"ephemeral\": 0, \"swap\": 0}, \"status\": \"active\", \"state\": \"running\", \"task_state\": \"\", \"image\": null, \"image_ref\": null, \"image_ref_url\": null, \"architecture\": \"x86_64\", \"os_type\": \"hvm\", \"vcpus\": 2, \"memory_mb\": 512, \"disk_gb\": 1, \"ephemeral_gb\": 0, \"root_gb\": 1, \"cpu_number\": 2}, \"message_id\": \"25286614-6b53-11eb-8d3e-525400c5aaec\", \"monotonic_time\": null, \"message_signature\": \"2acacbd9c515743bdde5ba54a4ed42d3ecd63573aa1a00faacabecf720123d3f\"}], \"timestamp\": \"2021-02-11 21:43:11.180978\"}"}, "context": {}}`

func genCollectdMessage() ([]byte, error) {
	metrics := []collectdMetric{
		{
			Values: []float64{
				rand.Float64() * 1000,
				rand.Float64() * 10000,
			},
			Dstypes: []string{
				"derive",
				"counter",
			},
			Dsnames: []string{
				"rx",
				"tx",
			},
			Host:           "controller-0.redhat.local",
			Time:           time.Now().Unix(),
			Interval:       5,
			Plugin:         "virt",
			PluginInstance: "asdf",
			Type:           "if_packets",
			TypeInstance:   "tap73125d-60",
		},
		{
			Values: []float64{
				rand.Float64() * 1000,
				rand.Float64() * 10000,
			},
			Dstypes: []string{
				"derive",
				"counter",
			},
			Dsnames: []string{
				"in",
				"out",
			},
			Host:           "controller-0.redhat.local",
			Time:           time.Now().Unix(),
			Interval:       5,
			Plugin:         "virt",
			PluginInstance: "asdf",
			Type:           "if_packets",
			TypeInstance:   "tap73125d-60",
		},
	}

	return json.Marshal(metrics)
}

type configT struct {
	Ceilometer bool
	Collectd   bool
	Interval   int
}

//DummyMetrics basic struct
type DummyMetrics struct {
	c configT
}

//Run implements type Transport
func (dm *DummyMetrics) Run(ctx context.Context, w transport.WriteFn, done chan bool) {

	for {
		select {
		case <-ctx.Done():
			goto done
		case <-time.After(time.Second * time.Duration(dm.c.Interval)):
			if dm.c.Ceilometer {
				w([]byte(ceilometerMessage))
			}
			if dm.c.Collectd {
				r, _ := genCollectdMessage()
				w(r)
			}
		}
	}

done:
}

//Listen ...
func (dm *DummyMetrics) Listen(e data.Event) {

}

//Config load configurations
func (dm *DummyMetrics) Config(c []byte) error {
	dm.c = configT{
		Ceilometer: true,
		Collectd:   true,
		Interval:   1,
	}
	err := config.ParseConfig(bytes.NewReader(c), &dm.c)
	if err != nil {
		return err
	}
	return nil
}

//New create new socket transport
func New(l *logging.Logger) transport.Transport {
	return &DummyMetrics{}
}
