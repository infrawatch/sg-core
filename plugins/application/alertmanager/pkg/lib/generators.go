package lib

import (
	"time"

	"github.com/infrawatch/sg-core/pkg/data"
)

const (
	alertSource     = "SmartGateway"
	isoTimeLayout   = "2006-01-02 15:04:05.000000"
	unknownSeverity = "unknown"
)

var (
	collectdAlertSeverity = map[string]string{
		"OKAY":    "info",
		"WARNING": "warning",
		"FAILURE": "critical",
	}
)

//GenerateAlert generate prometheus alert from event
func GenerateAlert(generatorURL string, event data.Event) PrometheusAlert {

	alert := PrometheusAlert{
		Labels:       make(map[string]string),
		Annotations:  make(map[string]string),
		GeneratorURL: generatorURL,
	}
	assimilateMap(event.Labels, &alert.Labels)
	assimilateMap(event.Annotations, &alert.Annotations)

	alert.Labels["alertname"] = event.Index
	alert.Labels["severity"] = event.Severity.String()
	alert.Labels["alertsource"] = alertSource
	alert.Labels["publisher"] = event.Publisher

	// set time to RFC3339
	// if zero allow alertmanager to set timestamp
	if event.Time != 0.0 {
		alert.StartsAt = time.Now().Format(time.RFC3339)
	}
	alert.SetSummary()
	return alert
}

func timeFromEpoch(epoch float64) string {
	whole := int64(epoch)
	t := time.Unix(whole, int64((float64(whole)-epoch)*1000000000))
	return t.Format(time.RFC3339)
}
