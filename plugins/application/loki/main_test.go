package main

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/infrawatch/apputils/connector/loki"
	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testConf = `
 connection: "http://localhost:3100"
 maxwaittime: 5s
 `
)

type lokiTestCase struct {
	Log    data.Event
	Result loki.LokiLog
}

var (
	testCases = []lokiTestCase{
		{
			Log: data.Event{
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
			Result: loki.LokiLog{
				LogMessage: "Supervising 0 threads of 0 processes of 0 users.",
				Timestamp:  time.Duration(1617888342) * time.Second,
				Labels: map[string]string{
					"host":     "localhost",
					"severity": "info",
					"facility": "daemon",
					"tag":      "rtkit-daemon[734691]",
					"source":   "rtkit-daemon",
					"file":     "",
					"cloud":    "cloud1",
					"region":   "<region-name>",
				},
			},
		},
		{
			Log: data.Event{
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
			Result: loki.LokiLog{
				LogMessage: "detected unhandled Python exception in 'interactive mode (python -c ...)'",
				Timestamp:  time.Duration(1620316105) * time.Second,
				Labels: map[string]string{
					"host":     "non-localhost",
					"severity": "info",
					"facility": "user",
					"tag":      "python3[804440]:",
					"source":   "python3",
					"file":     "",
					"cloud":    "cloud1",
					"region":   "Czech Republic",
				},
			},
		},
		{
			Log: data.Event{
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
			Result: loki.LokiLog{
				LogMessage: "Christmas!",
				Timestamp:  time.Duration(1640361600) * time.Second,
				Labels: map[string]string{
					"host":     "other-host",
					"severity": "critical",
					"facility": "authpriv",
					"tag":      "sudo[803493]:",
					"source":   "sudo",
					"file":     "santa.txt",
					"cloud":    "cloud1",
					"region":   "Home",
				},
			},
		},
	}
)

func TestLokiApp(t *testing.T) {
	tmpdir, err := ioutil.TempDir(".", "loki_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, logger.Destroy())
	}()

	t.Run("Test configuration", func(t *testing.T) {
		app := New(logger)
		err := app.Config([]byte(testConf))
		require.NoError(t, err)
	})

	t.Run("Test log message processing", func(t *testing.T) {
		results := make(chan interface{}, 100)
		app := &Loki{
			logger:     logger,
			logChannel: results,
		}

		for _, tstCase := range testCases {
			app.ReceiveEvent(tstCase.Log)
			res := <-results
			assert.Equal(t, tstCase.Result, res)
		}
	})
}
