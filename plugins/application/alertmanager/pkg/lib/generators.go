package lib

import (
	"time"

	"github.com/infrawatch/apputils/misc"

	"github.com/infrawatch/sg-core/pkg/data"
)

const (
	alertSource = "SmartGateway"
)

// GenerateAlert generate prometheus alert from event
func GenerateAlert(generatorURL string, event data.Event) PrometheusAlert {

	alert := PrometheusAlert{
		Labels:       make(map[string]string),
		Annotations:  make(map[string]string),
		GeneratorURL: generatorURL,
	}
	misc.AssimilateMap(event.Labels, &alert.Labels)
	misc.AssimilateMap(event.Annotations, &alert.Annotations)

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
