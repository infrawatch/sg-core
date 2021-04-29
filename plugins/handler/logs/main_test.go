package main

import (
	"testing"

	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type parsingTestCase struct {
	LogBlob   []byte
	ParsedLog data.Event
}

var testConfig = logConfig{
	MessageField:   "message",
	TimestampField: "@timestamp",
	HostnameField:  "host",
	SeverityField:  "severity",
}

var parsingCases = []parsingTestCase{
	{
		LogBlob: []byte(`{"@timestamp":"2021-04-08T15:25:42.604198+02:00", "host":"localhost", "severity":"7", "facility":"daemon", "tag":"rtkit-daemon[734691]", "source":"rtkit-daemon", "message":"Supervising 0 threads of 0 processes of 0 users.", "file":"", "cloud": "cloud1", "region": "<region-name>"}`),
		ParsedLog: data.Event{
			Index:     "logs-localhost-2021-4-8",
			Time:      1617888342,
			Type:      data.LOG,
			Publisher: "localhost",
			Severity:  data.INFO,
			Labels: map[string]interface{}{
				"host":     "localhost",
				"severity": "7",
				"facility": "daemon",
				"tag":      "rtkit-daemon[734691]",
				"source":   "rtkit-daemon",
				"file":     "",
				"cloud":    "cloud1",
				"region":   "<region-name>",
			},
			Message: "Supervising 0 threads of 0 processes of 0 users.",
		},
	},
	{
		LogBlob: []byte(`{"@timestamp":"2021-05-06T17:48:25.604198+02:00", "host":"non-localhost", "severity":"5", "facility":"user", "tag":"python3[804440]:", "source":"python3", "message":"detected unhandled Python exception in 'interactive mode (python -c ...)'", "file":"", "cloud": "cloud1", "region": "Czech Republic"}`),
		ParsedLog: data.Event{
			Index:     "logs-non-localhost-2021-5-6",
			Time:      1620316105,
			Type:      data.LOG,
			Publisher: "non-localhost",
			Severity:  data.INFO,
			Labels: map[string]interface{}{
				"host":     "non-localhost",
				"severity": "5",
				"facility": "user",
				"tag":      "python3[804440]:",
				"source":   "python3",
				"file":     "",
				"cloud":    "cloud1",
				"region":   "Czech Republic",
			},
			Message: "detected unhandled Python exception in 'interactive mode (python -c ...)'",
		},
	},
	{
		LogBlob: []byte(`{"@timestamp":"2021-12-24T18:00:00.000000+02:00", "host":"other-host", "severity":"1", "facility":"authpriv", "tag":"sudo[803493]:", "source":"sudo", "message":"Christmas!", "file":"santa.txt", "cloud": "cloud1", "region": "Home"}`),
		ParsedLog: data.Event{
			Index:     "logs-other-host-2021-12-24",
			Time:      1640361600,
			Type:      data.LOG,
			Publisher: "other-host",
			Severity:  data.CRITICAL,
			Labels: map[string]interface{}{
				"host":     "other-host",
				"severity": "1",
				"facility": "authpriv",
				"tag":      "sudo[803493]:",
				"source":   "sudo",
				"file":     "santa.txt",
				"cloud":    "cloud1",
				"region":   "Home",
			},
			Message: "Christmas!",
		},
	},
}

func TestLogs(t *testing.T) {
	t.Run("Test correct log parsing", func(t *testing.T) {
		for _, testCase := range parsingCases {
			// test parsing
			l := logHandler{
				config: testConfig,
			}
			parsed, err := l.parse(testCase.LogBlob)
			require.NoError(t, err)
			// test parsed content
			assert.Equal(t, testCase.ParsedLog, parsed)
			// test handling
			err = l.Handle(testCase.LogBlob, true, nil, func(evt data.Event) {
				assert.Equal(t, testCase.ParsedLog, evt)
			})
			require.NoError(t, err)
		}
	})

}
