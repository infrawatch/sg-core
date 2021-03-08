package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/application"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"

	"github.com/infrawatch/sg-core/plugins/application/alertmanager/pkg/lib"
)

const (
	appname = "alertmanager"
)

// AlertManager plugin suites for reporting alerts for Prometheus' alert manager
type AlertManager struct {
	configuration lib.AppConfig
	logger        *logging.Logger
	dump          chan lib.PrometheusAlert
}

// New constructor
func New(logger *logging.Logger) application.Application {
	return &AlertManager{
		configuration: lib.AppConfig{
			AlertManagerURL: "http://localhost",
			GeneratorURL:    "http://sg.localhost.localdomain",
		},
		logger: logger,
		dump:   make(chan lib.PrometheusAlert, 100),
	}
}

// ReceiveEvent is called whenever an event is broadcast on the event bus. The order of arguments
func (am *AlertManager) ReceiveEvent(event data.Event) {
	switch event.Type {
	case data.ERROR:
		//TODO: error handling
	case data.EVENT:
		// generate alert
		am.dump <- lib.GenerateAlert(am.configuration.GeneratorURL, event)
	case data.RESULT:
		//TODO: result type handling
	case data.LOG:
		//TODO: log handling
	case data.TASK:
	}

}

//Run implements main process of the application
func (am *AlertManager) Run(ctx context.Context, done chan bool) {
	wg := sync.WaitGroup{}

	for {
		select {
		case <-ctx.Done():
			goto done
		case dumped := <-am.dump:
			wg.Add(1)
			go func(dumped lib.PrometheusAlert, wg *sync.WaitGroup) {
				defer wg.Done()
				alert, err := json.Marshal(dumped)
				if err != nil {
					am.logger.Metadata(logging.Metadata{"plugin": appname, "alert": dumped})
					am.logger.Warn("failed to marshal alert - disregarding")
				} else {
					buff := bytes.NewBufferString("[")
					buff.Write(alert)
					buff.WriteString("]")

					req, err := http.NewRequest("POST", am.configuration.AlertManagerURL, buff)
					if err != nil {
						am.logger.Metadata(logging.Metadata{"plugin": appname, "error": err})
						am.logger.Error("failed to create http request")
					}
					req = req.WithContext(ctx)
					req.Header.Set("X-Custom-Header", "smartgateway")
					req.Header.Set("Content-Type", "application/json")

					client := &http.Client{}
					resp, err := client.Do(req)
					if err != nil {
						am.logger.Metadata(logging.Metadata{"plugin": appname, "error": err, "alert": buff.String()})
						am.logger.Error("failed to report alert to AlertManager")
					} else if resp.StatusCode != http.StatusOK {
						// https://github.com/prometheus/alertmanager/blob/master/api/v2/openapi.yaml#L170
						body, _ := ioutil.ReadAll(resp.Body)
						resp.Body.Close()
						am.logger.Metadata(logging.Metadata{
							"plugin": appname,
							"status": resp.Status,
							"header": resp.Header,
							"body":   string(body)})
						am.logger.Error("failed to report alert to AlertManager")
					}
				}
			}(dumped, &wg)
		}
	}

done:
	wg.Wait()
	am.logger.Metadata(logging.Metadata{"plugin": appname})
	am.logger.Info("exited")
}

//Config implements application.Application
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
