package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/application"
	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"

	"github.com/infrawatch/sg-core/plugins/application/alertmanager/pkg/lib"
)

const (
	appname = "alertmanager"
	workers = 10
)

// AlertManager plugin suites for reporting alerts for Prometheus' alert manager
type AlertManager struct {
	configuration lib.AppConfig
	logger        *logging.Logger
	dump          chan lib.PrometheusAlert
	client        *http.Client
}

// New constructor
func New(logger *logging.Logger, sendEvent bus.EventPublishFunc) application.Application {
	return &AlertManager{
		configuration: lib.AppConfig{
			AlertManagerURL: "http://localhost",
			GeneratorURL:    "http://sg.localhost.localdomain",
		},
		logger: logger,
		dump:   make(chan lib.PrometheusAlert, 100),
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// ReceiveEvent is called whenever an event is broadcast on the event bus. The order of arguments
func (am *AlertManager) ReceiveEvent(event data.Event) {
	switch event.Type {
	case data.ERROR:
		// TODO: error handling
	case data.EVENT:
		// generate alert
		am.dump <- lib.GenerateAlert(am.configuration.GeneratorURL, event)
	case data.RESULT:
		// TODO: result type handling
	case data.LOG:
		// TODO: log handling
	case data.TASK:
	}

}

// Run implements main process of the application
func (am *AlertManager) Run(ctx context.Context, done chan bool) {
	wg := sync.WaitGroup{}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case dumped := <-am.dump:
					alert, err := json.Marshal(dumped)
					if err != nil {
						am.logger.Metadata(logging.Metadata{"plugin": appname, "alert": dumped})
						am.logger.Warn("failed to marshal alert - disregarding")
						continue
					}

					buff := bytes.NewBufferString("[")
					buff.Write(alert)
					buff.WriteString("]")

					req, err := http.NewRequest("POST", am.configuration.AlertManagerURL, buff)
					if err != nil {
						am.logger.Metadata(logging.Metadata{"plugin": appname, "error": err})
						am.logger.Error("failed to create http request")
						continue
					}
					req = req.WithContext(ctx)
					req.Header.Set("X-Custom-Header", "smartgateway")
					req.Header.Set("Content-Type", "application/json")

					resp, err := am.client.Do(req)
					if err != nil {
						am.logger.Metadata(logging.Metadata{"plugin": appname, "error": err, "alert": buff.String()})
						am.logger.Error("failed to report alert to AlertManager")
						continue
					} else if resp.StatusCode != http.StatusOK {
						// https://github.com/prometheus/alertmanager/blob/master/api/v2/openapi.yaml#L170
						body, _ := io.ReadAll(resp.Body)
						am.logger.Metadata(logging.Metadata{
							"plugin": appname,
							"status": resp.Status,
							"header": resp.Header,
							"body":   string(body)})
						am.logger.Error("failed to report alert to AlertManager")
					}
					resp.Body.Close()
				}
			}
		}()
	}
	wg.Wait()
	am.logger.Metadata(logging.Metadata{"plugin": appname})
	am.logger.Info("exited")
}

// Config implements application.Application
func (am *AlertManager) Config(c []byte) error {
	am.configuration = lib.AppConfig{
		AlertManagerURL: "http://localhost",
		GeneratorURL:    "http://sg.localhost.localdomain",
	}
	err := config.ParseConfig(bytes.NewReader(c), &am.configuration)
	if err != nil {
		return err
	}
	return nil
}
