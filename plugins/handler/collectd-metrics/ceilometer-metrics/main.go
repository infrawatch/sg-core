package main

import (
	"context"

	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/handler"
)

// example message
/*
{
  "request": {
    "oslo.version": "2.0",
    "oslo.message": "{\"message_id\": \"36ce5025-85b4-4dcf-bce1-757cb96acf90\", \"publisher_id\": \"telemetry.publisher.controller-0.redhat.local\", \"event_type\": \"metering\", \"priority\": \"SAMPLE\", \"payload\": [{\"source\": \"openstack\", \"counter_name\": \"network.outgoing.packets\", \"counter_type\": \"cumulative\", \"counter_unit\": \"packet\", \"counter_volume\": 14, \"user_id\": \"581d4d733fad4baebd0edecb5dc6b889\", \"project_id\": \"40390761eaf7414c8125917efc21024c\", \"resource_id\": \"instance-00000001-13d97b93-63c6-4518-a885-05b95199ae9d-tapd042b21c-a9\", \"timestamp\": \"2021-02-08T20:45:28.286586\", \"resource_metadata\": {\"display_name\": \"BLUE\", \"name\": \"tapd042b21c-a9\", \"instance_id\": \"13d97b93-63c6-4518-a885-05b95199ae9d\", \"instance_type\": \"m1.tiny\", \"host\": \"34a0c5f56e8cc42025727379a8640387bb6c8e37b916b82940354c42\", \"instance_host\": \"compute-1.redhat.local\", \"flavor\": {\"id\": \"beeaf8e5-2bda-4b3c-88ad-cf41ad4df149\", \"name\": \"m1.tiny\", \"vcpus\": 2, \"ram\": 512, \"disk\": 1, \"ephemeral\": 0, \"swap\": 0}, \"status\": \"active\", \"state\": \"running\", \"task_state\": \"\", \"image\": null, \"image_ref\": null, \"image_ref_url\": null, \"architecture\": \"x86_64\", \"os_type\": \"hvm\", \"vcpus\": 2, \"memory_mb\": 512, \"disk_gb\": 1, \"ephemeral_gb\": 0, \"root_gb\": 1, \"mac\": \"fa:16:3e:92:a3:86\", \"fref\": null, \"parameters\": {\"interfaceid\": \"d042b21c-a92d-41cb-bc73-cb906cc151e7\", \"bridge\": \"br-int\"}, \"vnic_name\": \"tapd042b21c-a9\"}, \"message_id\": \"93b28638-6a4e-11eb-8ea0-52540021b407\", \"monotonic_time\": null, \"message_signature\": \"9db3fe4203b30f20073fd437f1e5b148faafe7029670a0cd7149657611233ce4\"}], \"timestamp\": \"2021-02-08 21:23:06.418913\"}"
  },
  "context": {}
}
*/

type ceilometerMetrics struct {
	Publisher string
	Payload   map[string]interface{}
}

type ceilometerMetricHandler struct {
}

func (c *ceilometerMetricHandler) Run(ctx context.Context, pf bus.MetricPublishFunc) {

}

func (c *ceilometerMetricHandler) Handle(blob []byte, pf bus.MetricPublishFunc) {

}

func New() handler.MetricHandler {
	return &ceilometerMetricHandler{}
}
