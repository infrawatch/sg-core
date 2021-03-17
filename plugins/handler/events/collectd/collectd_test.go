package collectd

import (
	"testing"

	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/stretchr/testify/assert"
)

type parsingTestCase struct {
	EventBlob []byte
	Sanitized string
	Event     data.Event
}

var parsingCases = []parsingTestCase{
	{
		EventBlob: []byte("[{\"labels\":{\"alertname\":\"collectd_connectivity_gauge\",\"instance\":\"d60b3c68f23e\",\"connectivity\":\"eno2\"," +
			"\"type\":\"interface_status\",\"severity\":\"FAILURE\",\"service\":\"collectd\"},\"annotations\":{\"summary\":\"\"," +
			"\"ves\":\"{\\\"domain\\\":\\\"stateChange\\\",\\\"eventId\\\":2,\\\"eventName\\\":\\\"interface eno2 up\\\"," +
			"\\\"lastEpochMicrosec\\\":1518790014024924,\\\"priority\\\":\\\"high\\\",\\\"reportingEntityName\\\":\\\"collectd connectivity plugin\\\"," +
			"\\\"sequence\\\":0,\\\"sourceName\\\":\\\"eno2\\\",\\\"startEpochMicrosec\\\":1518790009881440,\\\"version\\\":1.0," +
			"\\\"stateChangeFields\\\":{\\\"newState\\\":\\\"outOfService\\\",\\\"oldState\\\":\\\"inService\\\",\\\"stateChangeFieldsVersion\\\":1.0," +
			"\\\"stateInterface\\\":\\\"eno2\\\"}}\"},\"startsAt\":\"2018-02-16T14:06:54.024856417Z\"}]"),
		Sanitized: "[{\"labels\":{\"alertname\":\"collectd_connectivity_gauge\",\"instance\":\"d60b3c68f23e\",\"connectivity\":\"eno2\"," +
			"\"type\":\"interface_status\",\"severity\":\"FAILURE\",\"service\":\"collectd\"},\"annotations\":{\"summary\":\"\"," +
			"\"ves\":{\"domain\":\"stateChange\",\"eventId\":2,\"eventName\":\"interface eno2 up\",\"lastEpochMicrosec\":1518790014024924," +
			"\"priority\":\"high\",\"reportingEntityName\":\"collectd connectivity plugin\",\"sequence\":0,\"sourceName\":\"eno2\"," +
			"\"startEpochMicrosec\":1518790009881440,\"version\":1.0,\"stateChangeFields\":{\"newState\":\"outOfService\",\"oldState\":\"inService\"," +
			"\"stateChangeFieldsVersion\":1.0,\"stateInterface\":\"eno2\"}}},\"startsAt\":\"2018-02-16T14:06:54.024856417Z\"}]",
		Event: data.Event{
			Index:     "collectd_connectivity",
			Time:      1518790014,
			Type:      1,
			Publisher: "d60b3c68f23e",
			Severity:  3,
			Labels: map[string]interface{}{
				"alertname":    "collectd_connectivity_gauge",
				"connectivity": "eno2",
				"instance":     "d60b3c68f23e",
				"service":      "collectd",
				"severity":     "FAILURE",
				"type":         "interface_status",
			},
			Annotations: map[string]interface{}{
				"processed_by": "sg",
				"source_type":  "collectd",
				"summary":      "",
				"ves": map[string]interface{}{
					"domain":              "stateChange",
					"eventId":             float64(2),
					"eventName":           "interface eno2 up",
					"lastEpochMicrosec":   float64(1518790014024924),
					"priority":            "high",
					"reportingEntityName": "collectd connectivity plugin",
					"sequence":            float64(0),
					"sourceName":          "eno2",
					"startEpochMicrosec":  float64(1518790009881440),
					"stateChangeFields": map[string]interface{}{
						"newState":                 "outOfService",
						"oldState":                 "inService",
						"stateChangeFieldsVersion": float64(1),
						"stateInterface":           "eno2",
					},
					"version": float64(1),
				},
			},
			Message: "",
		},
	},
	{
		EventBlob: []byte("[{\"labels\":{\"alertname\":\"collectd_procevent_gauge\",\"instance\":\"d60b3c68f23e\",\"procevent\":\"bla.py\",\"type\":\"process_status\"," +
			"\"severity\":\"FAILURE\",\"service\":\"collectd\"},\"annotations\":{\"summary\":\"\",\"ves\":\"{\\\"domain\\\":\\\"fault\\\"," +
			"\\\"eventId\\\":3,\\\"eventName\\\":\\\"process bla.py (8537) down\\\",\\\"lastEpochMicrosec\\\":1518791119579620," +
			"\\\"priority\\\":\\\"high\\\",\\\"reportingEntityName\\\":\\\"collectd procevent plugin\\\",\\\"sequence\\\":0," +
			"\\\"sourceName\\\":\\\"bla.py\\\",\\\"startEpochMicrosec\\\":1518791111336973,\\\"version\\\":1.0,\\\"faultFields\\\":{" +
			"\\\"alarmCondition\\\":\\\"process bla.py (8537) state change\\\",\\\"alarmInterfaceA\\\":\\\"bla.py\\\"," +
			"\\\"eventSeverity\\\":\\\"CRITICAL\\\",\\\"eventSourceType\\\":\\\"process\\\",\\\"faultFieldsVersion\\\":1.0," +
			"\\\"specificProblem\\\":\\\"process bla.py (8537) down\\\",\\\"vfStatus\\\":\\\"Ready to terminate\\\"}}\"}," +
			"\"startsAt\":\"2018-02-16T14:25:19.579573212Z\"}]"),
		Sanitized: "[{\"labels\":{\"alertname\":\"collectd_procevent_gauge\",\"instance\":\"d60b3c68f23e\",\"procevent\":\"bla.py\",\"type\":\"process_status\"," +
			"\"severity\":\"FAILURE\",\"service\":\"collectd\"},\"annotations\":{\"summary\":\"\",\"ves\":{\"domain\":\"fault\",\"eventId\":3," +
			"\"eventName\":\"process bla.py (8537) down\",\"lastEpochMicrosec\":1518791119579620,\"priority\":\"high\",\"reportingEntityName\":\"collectd procevent plugin\"," +
			"\"sequence\":0,\"sourceName\":\"bla.py\",\"startEpochMicrosec\":1518791111336973,\"version\":1.0,\"faultFields\":{" +
			"\"alarmCondition\":\"process bla.py (8537) state change\",\"alarmInterfaceA\":\"bla.py\",\"eventSeverity\":\"CRITICAL\"," +
			"\"eventSourceType\":\"process\",\"faultFieldsVersion\":1.0,\"specificProblem\":\"process bla.py (8537) down\"," +
			"\"vfStatus\":\"Ready to terminate\"}}},\"startsAt\":\"2018-02-16T14:25:19.579573212Z\"}]",
		Event: data.Event{
			Index:     "collectd_procevent",
			Time:      1518791119,
			Type:      1,
			Publisher: "d60b3c68f23e",
			Severity:  3,
			Labels: map[string]interface{}{
				"alertname": "collectd_procevent_gauge",
				"instance":  "d60b3c68f23e",
				"procevent": "bla.py",
				"service":   "collectd",
				"severity":  "FAILURE",
				"type":      "process_status",
			},
			Annotations: map[string]interface{}{
				"processed_by": "sg",
				"source_type":  "collectd",
				"summary":      "",
				"ves": map[string]interface{}{
					"domain":    "fault",
					"eventId":   float64(3),
					"eventName": "process bla.py (8537) down",
					"faultFields": map[string]interface{}{
						"alarmCondition":     "process bla.py (8537) state change",
						"alarmInterfaceA":    "bla.py",
						"eventSeverity":      "CRITICAL",
						"eventSourceType":    "process",
						"faultFieldsVersion": float64(1),
						"specificProblem":    "process bla.py (8537) down",
						"vfStatus":           "Ready to terminate",
					},
					"lastEpochMicrosec":   float64(1518791119579620),
					"priority":            "high",
					"reportingEntityName": "collectd procevent plugin",
					"sequence":            float64(0),
					"sourceName":          "bla.py",
					"startEpochMicrosec":  float64(1518791111336973),
					"version":             float64(1),
				},
			},
			Message: "",
		},
	},
	{
		EventBlob: []byte(`[{"labels":{"alertname":"collectd_interface_if_octets","instance":"localhost.localdomain","interface":"lo","severity":"FAILURE",` +
			`"service":"collectd"},"annotations":{"summary":"Host localhost.localdomain, plugin interface (instance lo) type if_octets: ` +
			`Data source \"rx\" is currently 43596.224329. That is above the failure threshold of 0.000000.","DataSource":"rx",` +
			`"CurrentValue":"43596.2243286703","WarningMin":"nan","WarningMax":"nan","FailureMin":"nan","FailureMax":"0"},` +
			`"startsAt":"2019-09-18T21:11:19.281603240Z"}]`),
		Sanitized: "[{\"labels\":{\"alertname\":\"collectd_interface_if_octets\",\"instance\":\"localhost.localdomain\",\"interface\":\"lo\",\"severity\":\"FAILURE\"," +
			"\"service\":\"collectd\"},\"annotations\":{\"summary\":\"Host localhost.localdomain, plugin interface (instance lo) type if_octets: " +
			"Data source \\\"rx\\\" is currently 43596.224329. That is above the failure threshold of 0.000000.\",\"DataSource\":\"rx\"," +
			"\"CurrentValue\":\"43596.2243286703\",\"WarningMin\":\"nan\",\"WarningMax\":\"nan\",\"FailureMin\":\"nan\",\"FailureMax\":\"0\"}," +
			"\"startsAt\":\"2019-09-18T21:11:19.281603240Z\"}]",
		Event: data.Event{
			Index:     "collectd_interface_if",
			Time:      1.568841079e+09,
			Type:      1,
			Publisher: "localhost.localdomain",
			Severity:  3,
			Labels: map[string]interface{}{
				"alertname": "collectd_interface_if_octets",
				"instance":  "localhost.localdomain",
				"interface": "lo",
				"service":   "collectd",
				"severity":  "FAILURE",
			},
			Annotations: map[string]interface{}{
				"CurrentValue": "43596.2243286703",
				"DataSource":   "rx",
				"FailureMax":   "0",
				"FailureMin":   "nan",
				"WarningMax":   "nan",
				"WarningMin":   "nan",
				"processed_by": "sg",
				"source_type":  "collectd",
				"summary": "Host localhost.localdomain, plugin interface (instance lo) type if_octets: " +
					"Data source \"rx\" is currently 43596.224329. That is above the failure threshold of 0.000000.",
			},
			Message: "",
		},
	},
	{
		EventBlob: []byte(`[{"labels":{"alertname":"collectd_ovs_events_gauge","instance":"nfvha-comp-03","ovs_events":"br0","type":"link_status","severity":"OKAY",` +
			`"service":"collectd"},"annotations":{"summary":"link state of \"br0\" interface has been changed to \"UP\"",` +
			`"uuid":"c52f2aca-3cb1-48e3-bba3-100b54303d84"},"startsAt":"2018-02-22T20:12:19.547955618Z"}]`),
		Sanitized: "[{\"labels\":{\"alertname\":\"collectd_ovs_events_gauge\",\"instance\":\"nfvha-comp-03\",\"ovs_events\":\"br0\",\"type\":\"link_status\"," +
			"\"severity\":\"OKAY\",\"service\":\"collectd\"},\"annotations\":{\"summary\":\"link state of \\\"br0\\\" interface has been changed to \\\"UP\\\"\"," +
			"\"uuid\":\"c52f2aca-3cb1-48e3-bba3-100b54303d84\"},\"startsAt\":\"2018-02-22T20:12:19.547955618Z\"}]",
		Event: data.Event{
			Index:     "collectd_ovs_events",
			Time:      1519330339,
			Type:      1,
			Publisher: "nfvha-comp-03",
			Severity:  1,
			Labels: map[string]interface{}{
				"alertname":  "collectd_ovs_events_gauge",
				"instance":   "nfvha-comp-03",
				"ovs_events": "br0",
				"service":    "collectd",
				"severity":   "OKAY",
				"type":       "link_status",
			},
			Annotations: map[string]interface{}{
				"processed_by": "sg",
				"source_type":  "collectd",
				"summary":      "link state of \"br0\" interface has been changed to \"UP\"",
				"uuid":         "c52f2aca-3cb1-48e3-bba3-100b54303d84",
			},
			Message: "",
		},
	},
	{
		EventBlob: []byte(`{"labels":{"check":"elastic-check","client":"wubba.lubba.dub.dub.redhat.com","severity":"FAILURE"},"annotations":` +
			`{"command":"podman ps | grep elastic || exit 2","duration":0.043278607,"executed":1601900769,"issued":1601900769,` +
			`"output":"time=\"2020-10-05T14:26:09+02:00\" level=error msg=\"cannot mkdir /run/user/0/libpod: mkdir /run/user/0/libpod: permission denied\"\n",` +
			`"status":2,"ves":"{\"commonEventHeader\":{\"domain\":\"heartbeat\",\"eventType\":\"checkResult\",` +
			`\"eventId\":\"wubba.lubba.dub.dub.redhat.com-elastic-check\",\"priority\":\"High\",\"reportingEntityId\":\"918e8d04-c5ae-4e20-a763-8eb4f1af7c80\",` +
			`\"reportingEntityName\":\"wubba.lubba.dub.dub.redhat.com\",\"sourceId\":\"918e8d04-c5ae-4e20-a763-8eb4f1af7c80\",` +
			`\"sourceName\":\"wubba.lubba.dub.dub.redhat.com-collectd-sensubility\",\"startingEpochMicrosec\":1601900769,\"lastEpochMicrosec\":1601900769},` +
			`\"heartbeatFields\":{\"additionalFields\":{\"check\":\"elastic-check\",\"command\":\"podman ps | grep elastic || exit 2 || $0\",` +
			`\"duration\":\"0.043279\",\"executed\":\"1601900769\",\"issued\":\"1601900769\",\"output\":\"time=\\\"2020-10-05T14:26:09+02:00\\\" ` +
			`level=error msg=\\\"cannot mkdir /run/user/0/libpod: mkdir /run/user/0/libpod: permission denied\\\"\\n\",\"status\":\"2\"}}}"},` +
			`"startsAt":"2020-10-05T14:26:09+02:00"}`),
		Sanitized: "[{\"labels\":{\"check\":\"elastic-check\",\"client\":\"wubba.lubba.dub.dub.redhat.com\",\"severity\":\"FAILURE\"},\"annotations\":" +
			"{\"command\":\"podman ps | grep elastic || exit 2\",\"duration\":0.043278607,\"executed\":1601900769,\"issued\":1601900769," +
			"\"output\":\"time=\\\"2020-10-05T14:26:09+02:00\\\" level=error msg=\\\"cannot mkdir /run/user/0/libpod: mkdir /run/user/0/libpod: permission denied\\\"\\n\"," +
			"\"status\":2,\"ves\":{\"commonEventHeader\":{\"domain\":\"heartbeat\",\"eventType\":\"checkResult\",\"eventId\":\"wubba.lubba.dub.dub.redhat.com-elastic-check\"," +
			"\"priority\":\"High\",\"reportingEntityId\":\"918e8d04-c5ae-4e20-a763-8eb4f1af7c80\",\"reportingEntityName\":\"wubba.lubba.dub.dub.redhat.com\"," +
			"\"sourceId\":\"918e8d04-c5ae-4e20-a763-8eb4f1af7c80\",\"sourceName\":\"wubba.lubba.dub.dub.redhat.com-collectd-sensubility\",\"startingEpochMicrosec\":1601900769," +
			"\"lastEpochMicrosec\":1601900769},\"heartbeatFields\":{\"additionalFields\":{\"check\":\"elastic-check\",\"command\":\"podman ps | grep elastic || exit 2 || $0\"," +
			"\"duration\":\"0.043279\",\"executed\":\"1601900769\",\"issued\":\"1601900769\",\"output\":\"time=\\\"2020-10-05T14:26:09+02:00\\\" level=error " +
			"msg=\\\"cannot mkdir /run/user/0/libpod: mkdir /run/user/0/libpod: permission denied\\\"\\\\n\",\"status\":\"2\"}}}},\"startsAt\":\"2020-10-05T14:26:09+02:00\"}]",
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
				"duration":     float64(0.043278607),
				"executed":     float64(1601900769),
				"issued":       float64(1601900769),
				"output":       "time=\"2020-10-05T14:26:09+02:00\" level=error msg=\"cannot mkdir /run/user/0/libpod: mkdir /run/user/0/libpod: permission denied\"\n",
				"processed_by": "sg",
				"source_type":  "collectd",
				"status":       float64(2),
				"ves": map[string]interface{}{
					"commonEventHeader": map[string]interface{}{
						"domain":                "heartbeat",
						"eventId":               "wubba.lubba.dub.dub.redhat.com-elastic-check",
						"eventType":             "checkResult",
						"lastEpochMicrosec":     float64(1601900769),
						"priority":              "High",
						"reportingEntityId":     "918e8d04-c5ae-4e20-a763-8eb4f1af7c80",
						"reportingEntityName":   "wubba.lubba.dub.dub.redhat.com",
						"sourceId":              "918e8d04-c5ae-4e20-a763-8eb4f1af7c80",
						"sourceName":            "wubba.lubba.dub.dub.redhat.com-collectd-sensubility",
						"startingEpochMicrosec": float64(1601900769),
					},
					"heartbeatFields": map[string]interface{}{
						"additionalFields": map[string]interface{}{
							"check":    "elastic-check",
							"command":  "podman ps | grep elastic || exit 2 || $0",
							"duration": "0.043279",
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
	},
}

func TestCeilometerEvents(t *testing.T) {
	t.Run("Test correct event parsing.", func(t *testing.T) {
		for _, testCase := range parsingCases {
			// test sanitizing
			assert.Equal(t, testCase.Sanitized, sanitize(testCase.EventBlob))
			// test parsing
			coll := Collectd{}
			err := coll.Parse(testCase.EventBlob)
			assert.NoError(t, err)
			// test publishing
			expected := testCase.Event
			coll.PublishEvents(func(evt data.Event) {
				assert.Equal(t, expected, evt)
			})
		}
	})

}
