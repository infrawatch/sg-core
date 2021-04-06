package lib

import (
	"testing"
	"time"

	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/stretchr/testify/assert"
)

type generatorTestCase struct {
	Event data.Event
	Alert PrometheusAlert
}

var testCases = []generatorTestCase{
	{
		Event: data.Event{
			Index:     "ceilometer_image",
			Time:      1583504009,
			Type:      data.EVENT,
			Publisher: "telemetry.publisher.controller-0.redhat.local",
			Severity:  data.INFO,
			Labels: map[string]interface{}{
				"service":     "image.localhost",
				"project_id":  "0f500647077b47f08a8ca9181e9b7aef",
				"user_id":     "0f500647077b47f08a8ca9181e9b7aef",
				"resource_id": "c4f7e00b-df85-4b77-9e1a-26a1de4d5735",
				"name":        "cirros",
				"status":      "deleted",
				"created_at":  "2020-03-06T14:01:07",
				"deleted_at":  "2020-03-06T14:13:29",
				"size":        1.3287936e+07,
			},
			Annotations: map[string]interface{}{
				"source_type":  "ceilometer",
				"processed_by": "sg",
			},
		},
		Alert: PrometheusAlert{
			Labels: map[string]string{
				"service":     "image.localhost",
				"project_id":  "0f500647077b47f08a8ca9181e9b7aef",
				"user_id":     "0f500647077b47f08a8ca9181e9b7aef",
				"resource_id": "c4f7e00b-df85-4b77-9e1a-26a1de4d5735",
				"name":        "cirros",
				"status":      "deleted",
				"alertname":   "ceilometer_image",
				"alertsource": "SmartGateway",
				"publisher":   "telemetry.publisher.controller-0.redhat.local",
				"created_at":  "2020-03-06T14:01:07",
				"deleted_at":  "2020-03-06T14:13:29",
				"size":        "13287936.000000",
				"severity":    "info",
			},
			Annotations: map[string]string{
				"source_type":  "ceilometer",
				"processed_by": "sg",
				"summary":      "ceilometer image.localhost cirros info",
			},
			StartsAt:     time.Now().Format(time.RFC3339), // a bit unstable test, but meh
			GeneratorURL: "http://localhost",
		},
	},
	{
		Event: data.Event{
			Index:     "collectd_elastic_check",
			Time:      1601900769,
			Type:      1,
			Publisher: "unknown",
			Severity:  3,
			Labels: map[string]interface{}{
				"check":    "elastic-check",
				"client":   "wubba.lubba.dub.dub.redhat.com",
				"severity": "FAILURE",
			},
			Annotations: map[string]interface{}{
				"command":      "podman ps | grep elastic || exit 2",
				"duration":     int(1),
				"executed":     int(1601900769),
				"issued":       int(1601900769),
				"output":       "time=\"2020-10-05T14:26:09+02:00\" level=error msg=\"cannot mkdir /run/user/0/libpod: mkdir /run/user/0/libpod: permission denied\"\n",
				"processed_by": "sg",
				"source_type":  "collectd",
				"status":       int(2),
				"ves": map[string]interface{}{
					"commonEventHeader": map[string]interface{}{
						"domain":                "heartbeat",
						"eventId":               "wubba.lubba.dub.dub.redhat.com-elastic-check",
						"eventType":             "checkResult",
						"lastEpochMicrosec":     int(1601900769),
						"priority":              "High",
						"reportingEntityId":     "918e8d04-c5ae-4e20-a763-8eb4f1af7c80",
						"reportingEntityName":   "wubba.lubba.dub.dub.redhat.com",
						"sourceId":              "918e8d04-c5ae-4e20-a763-8eb4f1af7c80",
						"sourceName":            "wubba.lubba.dub.dub.redhat.com-collectd-sensubility",
						"startingEpochMicrosec": int(1601900769),
					},
					"heartbeatFields": map[string]interface{}{
						"additionalFields": map[string]interface{}{
							"check":    "elastic-check",
							"command":  "podman ps | grep elastic || exit 2",
							"duration": "1",
							"executed": "1601900769",
							"issued":   "1601900769",
							"output":   "time=\"2020-10-05T14:26:09+02:00\" level=error msg=\"cannot mkdir /run/user/0/libpod: mkdir /run/user/0/libpod: permission denied\"\n",
							"status":   "2",
						},
					},
				},
			},
			Message: "",
		},
		Alert: PrometheusAlert{
			Labels: map[string]string{
				"check":       "elastic-check",
				"client":      "wubba.lubba.dub.dub.redhat.com",
				"alertname":   "collectd_elastic_check",
				"alertsource": "SmartGateway",
				"publisher":   "unknown",
				"severity":    "critical",
			},
			Annotations: map[string]string{
				"command":               "podman ps | grep elastic || exit 2",
				"duration":              "1",
				"executed":              "1601900769",
				"issued":                "1601900769",
				"output":                "time=\"2020-10-05T14:26:09+02:00\" level=error msg=\"cannot mkdir /run/user/0/libpod: mkdir /run/user/0/libpod: permission denied\"\n",
				"processed_by":          "sg",
				"source_type":           "collectd",
				"status":                "2",
				"domain":                "heartbeat",
				"eventId":               "wubba.lubba.dub.dub.redhat.com-elastic-check",
				"eventType":             "checkResult",
				"lastEpochMicrosec":     "1601900769",
				"priority":              "High",
				"reportingEntityId":     "918e8d04-c5ae-4e20-a763-8eb4f1af7c80",
				"reportingEntityName":   "wubba.lubba.dub.dub.redhat.com",
				"sourceId":              "918e8d04-c5ae-4e20-a763-8eb4f1af7c80",
				"sourceName":            "wubba.lubba.dub.dub.redhat.com-collectd-sensubility",
				"startingEpochMicrosec": "1601900769",
				"check":                 "elastic-check",
				"summary":               "collectd heartbeat elastic-check critical",
			},
			StartsAt:     time.Now().Format(time.RFC3339), // a bit unstable test, but meh
			GeneratorURL: "http://localhost",
		},
	},
}

func TestGenerator(t *testing.T) {
	t.Run("Alert generation.", func(t *testing.T) {
		for _, tCase := range testCases {
			assert.EqualValues(t, tCase.Alert, GenerateAlert("http://localhost", tCase.Event))
		}
	})
}
