package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

//wrapper object for elasitcsearch index
type esIndex struct {
	index  string
	record []string
}

//Elasticsearch plugin saves events to Elasticsearch database
type Elasticsearch struct {
	configuration *lib.AppConfig
	logger        *logging.Logger
	client        *lib.Client
	store         chan esIndex
}

//New constructor
func New(logger *logging.Logger) application.Application {
	return &Elasticsearch{
		logger: logger,
		store:  make(chan esIndex, 100),
	}
}

//ReceiveEvent receive event from event bus
func (es *Elasticsearch) ReceiveEvent(hName string, eType data.EventType, msg string) {

	switch eType {
	case data.ERROR:
		//TODO: error handling
	case data.EVENT:
		// event handling
		if strings.HasSuffix(hName, handlersSuffix) {
			source := DataSource(0)
			if ok := source.SetFromString(hName[0:(len(hName) - len(handlersSuffix))]); !ok {
				es.logger.Metadata(logging.Metadata{"plugin": "elasticsearch", "source": source.String()})
				es.logger.Warn("received event from unknown data source - disregarding")
			} else {
				record := make(map[string]interface{})
				err := json.Unmarshal([]byte(hName), &record)
				if err != nil {
					es.logger.Metadata(logging.Metadata{"plugin": "elasticsearch", "event": msg, "error": err})
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
							es.store <- esIndex{
								index:  fmt.Sprintf("%s_events", source.String()),
								record: []string{string(rec)},
							}
						}
					}
				}
			}
		} else {
			es.logger.Metadata(logging.Metadata{"plugin": "elasticsearch", "event": msg})
			es.logger.Info("received unknown data in event bus - disregarding")
		}
	case data.RESULT:
		//TODO: sensubility result handling
	case data.LOG:
		//TODO: log collection handling
	}

}

//Run plugin process
func (es *Elasticsearch) Run(ctx context.Context, done chan bool) {
	if es.configuration.ResetIndex {
		supported := []string{}
		for i := range (DataSource(0)).ListAll() {
			supported = append(supported, DataSource(i).Prefix())
		}
		es.client.IndicesDelete(supported)
	}
	es.logger.Metadata(logging.Metadata{"plugin": "elasticsearch", "url": es.configuration.HostURL})
	es.logger.Info("storing events to Elasticsearch.")

	for {
		select {
		case <-ctx.Done():
			goto done
		case esi := <-es.store:
			if err := es.client.Index(esi.index, esi.record); err != nil {
				es.logger.Metadata(logging.Metadata{"plugin": "elasticsearch", "event": esi.record, "error": err})
				es.logger.Error("failed to index event - disregarding")
			} else {
				es.logger.Debug("successfully indexed document")
			}
		}
	}

done:
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
