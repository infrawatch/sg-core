package lib

import (
	"encoding/json"
	"time"

	"github.com/infrawatch/apputils/connector"
)

type LokiConfig struct {
	Connection  string `validate:"required"`
	BatchSize   int
	MaxWaitTime int
}

// temporary struct until we make a final decision on
// the event bus format.
type logFormat struct {
	Message   string
	Timestamp time.Time
	Tags      map[string]string
}

func CreateLokiLog(msg string) (connector.LokiLog, error) {
	var parsedLog logFormat
	err := json.Unmarshal([]byte(msg), &parsedLog)
	if err != nil {
		return connector.LokiLog{}, err
	}
	delete(parsedLog.Tags, "file")
	output := connector.LokiLog{
		Labels: parsedLog.Tags,
	}

	output.LogMessage = parsedLog.Message
	output.Timestamp = time.Duration(parsedLog.Timestamp.Unix()) * time.Second
	return output, nil
}


