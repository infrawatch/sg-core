package lib

import (
	"sort"
	"strings"
)

// PrometheusAlert represents data structure used for sending alerts to Prometheus Alert Manager
type PrometheusAlert struct {
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     string            `json:"startsAt,omitempty"`
	EndsAt       string            `json:"endsAt,omitempty"`
	GeneratorURL string            `json:"generatorURL"`
}

type surrogates []string

var (
	agentSurrogates = surrogates{"source_type"}
	eventSurrogates = surrogates{
		"eventSourceType",
		"type",
		"domain",
		"service",
	}
	entitySurrogates = surrogates{
		"name",
		"DataSource",
		"check",
		"connectivity",
		"procevent",
		"interface",
	}
	statusSurrogates = surrogates{"severity"}
)

// SetName generates unique name for the alert and creates new key/value pair for it in Labels
// Note: since this is not used, shouldn't it be deleted?
func (alert *PrometheusAlert) SetName() {
	if _, ok := alert.Labels["name"]; !ok {
		keys := make([]string, 0, len(alert.Labels))
		for k := range alert.Labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		values := make([]string, 0, len(alert.Labels)-1)
		for _, k := range keys {
			if k != "severity" {
				values = append(values, alert.Labels[k])
			}
		}
		alert.Labels["name"] = strings.Join(values, "_")
	}
}

// SetSummary generates summary annotation in case it is empty
func (alert *PrometheusAlert) SetSummary() {
	generate := false
	if _, ok := alert.Annotations["summary"]; ok {
		if alert.Annotations["summary"] == "" {
			generate = true
		}
	} else {
		generate = true
	}

	if generate {
		if val, ok := alert.Labels["summary"]; ok && alert.Labels["summary"] != "" {
			alert.Annotations["summary"] = val
		} else {
			values := make([]string, 0, 3)
			for _, surr := range []surrogates{agentSurrogates, eventSurrogates, entitySurrogates, statusSurrogates} {
				for _, key := range surr {
					if val, ok := alert.Labels[key]; ok {
						values = append(values, val)
						break
					} else if val, ok := alert.Annotations[key]; ok {
						values = append(values, val)
						break
					}
				}
			}
			alert.Annotations["summary"] = strings.Join(values, " ")
		}
	}
}
