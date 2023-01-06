package main

import (
	"context"
	"time"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/transport"
)

var msgBuffer []byte

var logMessages = []string{
	`", "host":"localhost", "severity":"7", "facility":"daemon", "tag":"rtkit-daemon[734691]:", "source":"rtkit-daemon", "message":"Supervising 0 threads of 0 processes of 0 users.", "file":"", "cloud": "cloud1", "region": "<region-name>"}`,
	`", "host":"localhost", "severity":"5", "facility":"user", "tag":"python3[804440]:", "source":"python3", "message":"detected unhandled Python exception in 'interactive mode (python -c ...)'", "file":"", "cloud": "cloud1", "region": "<region-name>"}`,
	`", "host":"localhost", "severity":"1", "facility":"authpriv", "tag":"sudo[803493]:", "source":"sudo", "message":"   jarda : 1 incorrect password attempt ; TTY=pts\/1 ; PWD=\/home\/jarda\/go\/src\/github.com\/vyzigold\/sg-core\/plugins\/application\/loki ; USER=root ; COMMAND=\/usr\/bin\/ls", "file":"", "cloud": "cloud1", "region": "<region-name>"}`,
}

// DummyLogs plugin struct
type DummyLogs struct {
	logger *logging.Logger
}

// Run implements type Transport
func (dl *DummyLogs) Run(ctx context.Context, wrFn transport.WriteFn, done chan bool) {

	for {
		for _, logEnding := range logMessages {
			t := time.Now()
			timestamp, err := t.MarshalText()
			if err != nil {
				dl.logger.Metadata(logging.Metadata{"plugin": "dummy-logs"})
				dl.logger.Warn("Failed to get current timestamp")
				continue
			}
			log := "{\"@timestamp\":\"" + string(timestamp) + logEnding
			select {
			case <-ctx.Done():
				goto done
			case <-time.After(time.Second * 1):
				time.Sleep(time.Second * 1)
				msgBuffer = []byte(log)
				wrFn(msgBuffer)
			}
		}
	}

done:
}

// Listen ...
func (dl *DummyLogs) Listen(e data.Event) {

}

// Config load configurations
func (dl *DummyLogs) Config(c []byte) error {
	return nil
}

// New create new socket transport
func New(l *logging.Logger) transport.Transport {
	return &DummyLogs{
		logger: l,
	}
}
