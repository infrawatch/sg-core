package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/application"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/pkg/errors"

	"github.com/infrawatch/sg-core/plugins/application/elasticsearch/pkg/lib"
)

const handlersSuffix = "-events"

//DataSource indentifies a format of incoming data in the message bus channel.
type DataSource int

//ListAll returns slice of supported data sources in human readable names.
func (src DataSource) ListAll() []string {
	return []string{"generic", "collectd", "ceilometer"}
}

//SetFromString resets value according to given human readable identification. Returns false if invalid identification was given.
func (src *DataSource) SetFromString(name string) bool {
	for index, value := range src.ListAll() {
		if name == value {
			*src = DataSource(index)
			return true
		}
	}
	return false
}

//String returns human readable data type identification.
func (src DataSource) String() string {
	return (src.ListAll())[src]
}

//Prefix returns human readable data type identification.
func (src DataSource) Prefix() string {
	return fmt.Sprintf("%s_*", src.String())
}

//Elasticsearch plugin saves events to Elasticsearch database
type Elasticsearch struct {
	configuration *lib.AppConfig
	logger        *logging.Logger
	client        *lib.Client
}

//New constructor
func New(logger *logging.Logger) application.Application {
	return &Elasticsearch{logger: logger}
}

//Run run scrape endpoint
func (es *Elasticsearch) Run(ctx context.Context, eChan chan data.Event, mChan chan []data.Metric, done chan bool) {
	if es.configuration.ResetIndex {
		supported := []string{}
		for i := range (DataSource(0)).ListAll() {
			supported = append(supported, DataSource(i).Prefix())
		}
		es.client.IndicesDelete(supported)
	}

	es.logger.Metadata(logging.Metadata{"plugin": "elasticsearch", "url": es.configuration.HostURL})
	es.logger.Info("Storing events to Elasticsearch.")

	for {
		select {
		case <-ctx.Done():
			goto done
		case event := <-eChan:
			switch event.Type {
			case 0:
				//TODO: error handling
			case 1:
				// event handling
				if strings.HasSuffix(event.Handler, handlersSuffix) {
					source := DataSource(0)
					if ok := source.SetFromString(event.Handler[0:(len(event.Handler) - len(handlersSuffix))]); !ok {
						es.logger.Metadata(logging.Metadata{"plugin": "elasticsearch", "source": source.String()})
						es.logger.Warn("received event from unknown data source - disregarding")
					} else {
						record := make(map[string]interface{})

						err := json.Unmarshal([]byte(event.Message), &record)
						if err != nil {
							es.logger.Metadata(logging.Metadata{"plugin": "elasticsearch", "event": record, "error": err})
							es.logger.Error("failed to unmarshal event - disregarding")
						} else {
							// format message if needed
							err := lib.EventFormatters[source.String()](record)
							if err != nil {
								es.logger.Metadata(logging.Metadata{"plugin": "elasticsearch", "event": record, "error": err})
								es.logger.Error("failed to format event - disregarding")
							} else {
								rec, err := json.Marshal(record)
								if err != nil {
									es.logger.Metadata(logging.Metadata{"plugin": "elasticsearch", "event": record, "error": err})
									es.logger.Error("failed to marshal event - disregarding")
								} else {
									if err = es.client.Index(fmt.Sprintf("%s_events", source.String()), []string{string(rec)}); err != nil {
										es.logger.Metadata(logging.Metadata{"plugin": "elasticsearch", "event": record, "error": err})
										es.logger.Error("failed to index event - disregarding")
									} else {
										es.logger.Debug("successfully indexed document")
									}
								}
							}
						}
					}
				} else {
					es.logger.Metadata(logging.Metadata{"plugin": "elasticsearch", "event": event})
					es.logger.Info("received unknown data in event bus - disregarding")
				}
			case 2:
				//TODO: sensubility result handling
			case 3:
				//TODO: log collection handling
			}
		case metrics := <-mChan:
			es.logger.Metadata(logging.Metadata{"plugin": "elasticsearch", "metrics": metrics})
			es.logger.Debug("received metric data - disregarding")
		}
	}

done:
	_, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	es.logger.Metadata(logging.Metadata{"plugin": "elasticsearch"})
	es.logger.Info("exited")
}

//Config implements application.Application
func (es *Elasticsearch) Config(c []byte) error {
	es.configuration = &lib.AppConfig{
		HostURL:       "",
		UseTLS:        false,
		TLSServerName: "",
		TLSClientCert: "",
		TLSClientKey:  "",
		TLSCaCert:     "",
		UseBasicAuth:  false,
		User:          "",
		Password:      "",
		ResetIndex:    false,
	}
	err := config.ParseConfig(bytes.NewReader(c), es.configuration)
	if err != nil {
		return err
	}

	es.client, err = lib.NewElasticClient(es.configuration)
	if err != nil {
		return errors.Wrap(err, "failed to connect to Elasticsearch host")
	}
	return nil
}
