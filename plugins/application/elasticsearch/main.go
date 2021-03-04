package main

import (
	"bytes"
	"context"
	"time"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/application"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"

	"github.com/infrawatch/sg-core/plugins/application/elasticsearch/pkg/lib"
)

const (
	appname           = "elasticsearch"
	genericSuffix     = "_generic"
	eventRecordFormat = `{"event_type":"%s","generated":"%s","severity":"%s","labels":%s,"annotations":%s}`
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

//wrapper object for elasitcsearch index
type esIndex struct {
	index  string
	record []string
}

//used to marshal event into es usable json
type record struct {
	EventType   string                 `json:"event_type"`
	Generated   string                 `json:"generated"`
	Severity    string                 `json:"severity"`
	Labels      map[string]interface{} `json:"labels"`
	Annotations map[string]interface{} `json:"annotations"`
}

//Elasticsearch plugin saves events to Elasticsearch database
type Elasticsearch struct {
	configuration *lib.AppConfig
	logger        *logging.Logger
	client        *lib.Client
	buffer        map[string][]string
	dump          chan esIndex
}

//New constructor
func New(logger *logging.Logger) application.Application {
	return &Elasticsearch{
		logger: logger,
		buffer: make(map[string][]string),
		dump:   make(chan esIndex, 100),
	}
}

//ReceiveEvent receive event from event bus
func (es *Elasticsearch) ReceiveEvent(event data.Event) {
	switch event.Type {
	case data.ERROR:
		//TODO: error handling
	case data.EVENT:
		// buffer or index record
		var recordList []string
		record, err := formatRecord(event)
		if err != nil {
			es.logger.Metadata(logging.Metadata{"plugin": appname, "event": event})
			es.logger.Error("failed formating record")
			return
		}
		if es.configuration.BufferSize > 1 {
			if _, ok := es.buffer[event.Index]; !ok {
				es.buffer[event.Index] = make([]string, 0, es.configuration.BufferSize)
			}

			es.buffer[event.Index] = append(es.buffer[event.Index], record)
			if len(es.buffer[event.Index]) < es.configuration.BufferSize {
				// buffer is not full, don't send
				es.logger.Metadata(logging.Metadata{"plugin": appname, "record": record})
				es.logger.Debug("buffering record")
				return
			}
			recordList = es.buffer[event.Index]
			delete(es.buffer, event.Index)
		} else {
			recordList = []string{record}
		}
		es.dump <- esIndex{index: event.Index, record: recordList}
	case data.RESULT:
		//TODO: result
	case data.LOG:
		//TODO: log
	}

}

//Run plugin process
func (es *Elasticsearch) Run(ctx context.Context, done chan bool) {
	es.logger.Metadata(logging.Metadata{"plugin": appname, "url": es.configuration.HostURL})
	es.logger.Info("storing events to Elasticsearch.")

	if es.configuration.ResetIndices != nil {
		err := es.client.IndicesDelete(es.configuration.ResetIndices)
		if err != nil {
			es.logger.Metadata(logging.Metadata{"plugin": appname, "error": err})
			es.logger.Error("failed removing indices")
			done <- true
			return
		}
		es.logger.Metadata(logging.Metadata{"plugin": appname, "indices": es.configuration.ResetIndices})
		es.logger.Info("removed indices")
	}

	for {
		select {
		case <-ctx.Done():
			goto done
		case dumped := <-es.dump:
			if err := es.client.Index(dumped.index, dumped.record, es.configuration.BulkIndex); err != nil {
				es.logger.Metadata(logging.Metadata{"plugin": appname, "event": dumped.record, "error": err})
				es.logger.Error("failed to index event - disregarding")
			} else {
				es.logger.Debug("successfully indexed document(s)")
			}
		}
	}

done:
	es.logger.Metadata(logging.Metadata{"plugin": appname})
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
		BufferSize:    1,
		BulkIndex:     false,
	}
	err := config.ParseConfig(bytes.NewReader(c), es.configuration)
	if err != nil {
		return err
	}

	if es.configuration.UseBasicAuth && !es.configuration.UseTLS {
		es.logger.Metadata(logging.Metadata{"plugin": appname})
		es.logger.Warn("insecure: using basic authentication without TLS enabled")
	}

	es.client, err = lib.NewElasticClient(es.configuration)
	if err != nil {
		return errors.Wrap(err, "failed to connect to Elasticsearch host")
	}
	return nil
}

func formatRecord(e data.Event) (string, error) {
	record := record{
		EventType:   e.Type.String(),
		Generated:   timeFromEpoch(e.Time),
		Severity:    e.Severity.String(),
		Labels:      e.Labels,
		Annotations: e.Annotations,
	}

	res, err := json.Marshal(record)
	if err != nil {
		return "", err
	}

	return string(res), nil
}

// Get time in RFC3339
func timeFromEpoch(epoch float64) string {
	if epoch == 0.0 {
		return time.Now().Format(time.RFC3339)
	}
	whole := int64(epoch)
	t := time.Unix(whole, int64((float64(whole)-epoch)*1000000000))
	return t.Format(time.RFC3339)
}
