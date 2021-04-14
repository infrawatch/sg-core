package lib

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/infrawatch/sg-core/pkg/data"
)

// SyslogSeverity holds received syslog severity and it's meaning
type SyslogSeverity int

// Human readable syslog severities
const (
	EMERGENCY SyslogSeverity = iota
	ALERT
	CRITICAL
	ERROR
	WARNING
	NOTICE
	INFORMATIONAL
	DEBUG
	UNKNOWN
)

const (
	corrSeparator  = " "
	corrTokenCount = 10
)

// String return text representation of severity
func (rs SyslogSeverity) String() string {
	return []string{
		"emergency",
		"alert",
		"critical",
		"error",
		"warning",
		"notice",
		"info",
		"debug",
		"unknown",
	}[rs]
}

func (rs *SyslogSeverity) fromMessage(msg string) bool {
	msgTokens := strings.SplitN(msg, corrSeparator, corrTokenCount)
	for i := EMERGENCY; i < UNKNOWN; i++ {
		rex, err := regexp.Compile(i.String())
		if err != nil {
			continue
		}
		for tid := range msgTokens {
			if msgTokens[tid] == "" {
				continue
			}
			if match := rex.FindStringIndex(strings.ToLower(msgTokens[tid])); match != nil {
				*rs = i
				return true
			}
		}
	}
	return false
}

// ToEventSeverity transforms syslog severity to appropriate sg severity.
func (rs SyslogSeverity) ToEventSeverity() data.EventSeverity {
	return []data.EventSeverity{
		data.CRITICAL,
		data.CRITICAL,
		data.CRITICAL,
		data.CRITICAL,
		data.WARNING,
		data.INFO,
		data.INFO,
		data.DEBUG,
		data.UNKNOWN,
	}[rs]
}

// GetSeverityFromLog returns appropriate SyslogSeverity value based on log values.
// Primarily SeverityField is used, but can be parsed from MessageField if CorrectSeverity is true.
func GetSeverityFromLog(log map[string]interface{}, config LogConfig) SyslogSeverity {
	severity := UNKNOWN
	if severitystring, ok := log[config.SeverityField].(string); ok {
		s, err := strconv.Atoi(severitystring)
		if err == nil {
			severity = SyslogSeverity(s)
		}
	}
	if config.CorrectSeverity {
		severity.fromMessage(log[config.MessageField].(string))
	}
	return severity
}
