package lib

import (
	"fmt"
	"time"

	"github.com/infrawatch/apputils/connector/loki"
	"github.com/infrawatch/apputils/misc"

	"github.com/infrawatch/sg-core/pkg/data"
)

// Creates labels used by Loki.
func createLabels(rawLabels map[string]interface{}) (map[string]string, error) {
	result := make(map[string]string)
	misc.AssimilateMap(rawLabels, &result)
	if len(result) == 0 {
		return nil, fmt.Errorf("unable to create log labels")
	}
	return result, nil
}

// CreateLokiLog forms event to a structure suitable for storage in Loki
func CreateLokiLog(log data.Event) (loki.LokiLog, error) {
	labels, err := createLabels(log.Labels)
	if err != nil {
		return loki.LokiLog{}, err
	}

	// correct severity value in labels
	labels["severity"] = log.Severity.String()

	output := loki.LokiLog{
		LogMessage: log.Message,
		Timestamp:  time.Duration(log.Time) * time.Second,
		Labels:     labels,
	}
	return output, nil
}
