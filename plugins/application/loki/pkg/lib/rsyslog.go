package lib

import (
	"encoding/json"
	"strconv"

	"github.com/infrawatch/apputils/connector"
	"github.com/infrawatch/apputils/logging"

	"fmt"
	"time"
)

var severityArray = []string{
	"Emergency",
	"Alert",
	"Critical",
	"Error",
	"Warning",
	"Notice",
	"Informational",
	"Debug",
}

//RsyslogLog represents message received from rsyslog source with following template:
//template (name="rsyslog-record" type="list" option.jsonf="on")
//{
//    property(format="jsonf" dateFormat="rfc3339" name="timereported" outname="@timestamp" )
//    property(format="jsonf" name="hostname" outname="host" )
//    property(format="jsonf" name="syslogseverity" outname="severity" )
//    property(format="jsonf" name="syslogfacility-text" outname="facility" )
//    property(format="jsonf" name="syslogtag" outname="tag" )
//    property(format="jsonf" name="app-name" outname="source" )
//    property(format="jsonf" name="msg" outname="message" )
//    property(format="jsonf" name="$!metadata!filename" outname="file")
//    constant(format="jsonf" value="<cloud-name>" outname="cloud")
//    constant(format="jsonf" value="<region-name>" outname="region")
//}
type RsyslogLog struct {
	Timestamp time.Time `json:"@timestamp"`
	Host      string    `json:"hostname"`
	Severity  string    `json:"severity"`
	Facility  string    `json:"facility"`
	Tag       string    `json:"tag"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
	File      string    `json:"file"`
	Cloud     string    `json:"cloud"`
	Region    string    `json:"region"`
}

//CreateLokiLog formats log from rsyslog handler for Loki
func (log *RsyslogLog) CreateLokiLog() (connector.LokiLog, error) {
	output := connector.LokiLog{
		Labels: map[string]string{
			"hostname": log.Host,
			"source":   log.Source,
			"cloud":    log.Cloud,
			"region":   log.Region,
			"file":     log.File,
		},
	}

	output.LogMessage = fmt.Sprintf("[%s] %s %s %s",
		log.Severity,
		log.Host,
		log.Tag,
		log.Message)
	output.Timestamp = time.Duration(log.Timestamp.Unix()) * time.Second
	return output, nil
}

func ParseRsyslog(input string, logger *logging.Logger) (*connector.LokiLog, error) {
	var rsysLog RsyslogLog
	err := json.Unmarshal([]byte(input), &rsysLog)
	if err != nil {
		logger.Metadata(map[string]interface{}{
			"error":   err,
			"message": input,
		})
		logger.Warn("Wrong log format received")
		return nil, err
	}
	sevNum, err := strconv.Atoi(rsysLog.Severity)
	if sevNum > 7 || sevNum < 0 {
		return nil, fmt.Errorf("Unknown severity number in received rsyslog message: %s", rsysLog.Severity)
	}
	rsysLog.Severity = severityArray[sevNum]

	log, err := rsysLog.CreateLokiLog()
	if err != nil {
		logger.Metadata(map[string]interface{}{
			"error": err,
		})
		logger.Error("Failed formatting rsyslog log to loki log")
	}
	return &log, err
}
