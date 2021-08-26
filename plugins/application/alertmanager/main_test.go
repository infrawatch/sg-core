package main

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/plugins/application/alertmanager/pkg/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testConf = `
alertManagerUrl: "http://127.0.0.1/test"
generatorUrl: "https://unit-test-sgcore.infrawatch"
`
)

type alertTestCase struct {
	Event  data.Event
	Result lib.PrometheusAlert
}

var (
	alertCases = []alertTestCase{
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
					"size":        int(13287936),
				},
				Annotations: map[string]interface{}{
					"source_type":  "ceilometer",
					"processed_by": "sg",
				},
			},
			Result: lib.PrometheusAlert{
				Labels: map[string]string{
					"alertname":   "ceilometer_image",
					"alertsource": "SmartGateway",
					"created_at":  "2020-03-06T14:01:07",
					"deleted_at":  "2020-03-06T14:13:29",
					"name":        "cirros",
					"project_id":  "0f500647077b47f08a8ca9181e9b7aef",
					"publisher":   "telemetry.publisher.controller-0.redhat.local",
					"resource_id": "c4f7e00b-df85-4b77-9e1a-26a1de4d5735",
					"service":     "image.localhost",
					"severity":    "info",
					"size":        "13287936",
					"status":      "deleted",
					"user_id":     "0f500647077b47f08a8ca9181e9b7aef",
				},
				Annotations: map[string]string{
					"processed_by": "sg",
					"source_type":  "ceilometer",
					"summary":      "ceilometer image.localhost cirros info",
				},
				StartsAt:     "2021-04-06T14:39:45+02:00",
				EndsAt:       "",
				GeneratorURL: "https://unit-test-sgcore.infrawatch",
			},
		},
		{
			Event: data.Event{
				Index:     "collectd_elastic_check",
				Time:      1601900769,
				Type:      1,
				Publisher: "unknown",
				Severity:  data.CRITICAL,
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
					"output":       "time=\"2020-10-05T14:26:09+02:00\" level=error msg=\"cannot mkdir /run/user/0/libpod: mkdir /run/user/0/libpod: permission denied\"\\n",
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
								"output":   "time=\"2020-10-05T14:26:09+02:00\" level=error msg=\"cannot mkdir /run/user/0/libpod: mkdir /run/user/0/libpod: permission denied\"\\n",
								"status":   "2",
							},
						},
					},
				},
				Message: "",
			},
			Result: lib.PrometheusAlert{
				Labels: map[string]string{
					"alertname":   "collectd_elastic_check",
					"alertsource": "SmartGateway",
					"check":       "elastic-check",
					"client":      "wubba.lubba.dub.dub.redhat.com",
					"publisher":   "unknown",
					"severity":    "critical",
				},
				Annotations: map[string]string{
					"check":                 "elastic-check",
					"command":               "podman ps | grep elastic || exit 2",
					"domain":                "heartbeat",
					"duration":              "1",
					"eventId":               "wubba.lubba.dub.dub.redhat.com-elastic-check",
					"eventType":             "checkResult",
					"executed":              "1601900769",
					"issued":                "1601900769",
					"lastEpochMicrosec":     "1601900769",
					"output":                "time=\"2020-10-05T14:26:09+02:00\" level=error msg=\"cannot mkdir /run/user/0/libpod: mkdir /run/user/0/libpod: permission denied\"\\n",
					"priority":              "High",
					"processed_by":          "sg",
					"reportingEntityId":     "918e8d04-c5ae-4e20-a763-8eb4f1af7c80",
					"reportingEntityName":   "wubba.lubba.dub.dub.redhat.com",
					"sourceId":              "918e8d04-c5ae-4e20-a763-8eb4f1af7c80",
					"sourceName":            "wubba.lubba.dub.dub.redhat.com-collectd-sensubility",
					"source_type":           "collectd",
					"startingEpochMicrosec": "1601900769",
					"status":                "2",
					"summary":               "collectd heartbeat elastic-check critical",
				},
				StartsAt:     "2021-04-06T14:39:45+02:00",
				EndsAt:       "",
				GeneratorURL: "https://unit-test-sgcore.infrawatch",
			},
		},
	}
)

func TestAlertmanagerApp(t *testing.T) {
	tmpdir, err := ioutil.TempDir(".", "alertman_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, logger.Destroy())
	}()

	t.Run("Test configuration", func(t *testing.T) {
		app := New(logger, bus.EventPublishFunc)
		err := app.Config([]byte(testConf))
		require.NoError(t, err)
	})

	t.Run("Test alert generation", func(t *testing.T) {
		results := make(chan lib.PrometheusAlert, len(alertCases))
		app := &AlertManager{
			logger: logger,
			dump:   results,
		}
		err := app.Config([]byte(testConf))
		require.NoError(t, err)

		for _, tstCase := range alertCases {
			app.ReceiveEvent(tstCase.Event)
			res := <-results
			assert.Equal(t, tstCase.Result.Labels, res.Labels)
			assert.Equal(t, tstCase.Result.Annotations, res.Annotations)
			assert.Equal(t, tstCase.Result.GeneratorURL, res.GeneratorURL)
		}
	})
}
