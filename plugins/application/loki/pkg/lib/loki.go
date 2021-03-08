package lib

import (
	"time"
	"fmt"

	"github.com/infrawatch/apputils/connector"
	"github.com/infrawatch/sg-core/pkg/data"
)

type LokiConfig struct {
	Connection  string `validate:"required"`
	BatchSize   int
	MaxWaitTime int
}

// Creates labels used by Loki.
func createLabels(rawLabels map[string]interface{}) (map[string]string, error) {
	result := make(map[string]string)
	assimilateMap(rawLabels, &result)
	if len(result) == 0 {
		return nil, fmt.Errorf("unable to create log labels")
	}
	return result, nil
}

func CreateLokiLog(log data.Event) (connector.LokiLog, error) {
	labels, err := createLabels(log.Labels)
	if err != nil {
		return connector.LokiLog{}, err
	}

	msg, ok := log.Annotations["log_message"].(string)
	if !ok {
		return connector.LokiLog{}, fmt.Errorf("unable to locate the log message")
	}

	output := connector.LokiLog {
		LogMessage: msg,
		Timestamp: time.Duration(time.Duration(log.Time) * time.Second),
		Labels: labels,
	}
	return output, nil
}


