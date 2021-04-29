package lib

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type alertTestCase struct {
	Alert   PrometheusAlert
	Summary string
}

var alertTestCases = []alertTestCase{
	{
		Alert: PrometheusAlert{
			Labels: map[string]string{
				"service":  "image.localhost",
				"name":     "cirros",
				"severity": "info",
			},
			Annotations: map[string]string{
				"source_type": "ceilometer",
				"summary":     "",
			},
		},
		Summary: "ceilometer image.localhost cirros info",
	},
	{
		Alert: PrometheusAlert{
			Labels: map[string]string{
				"check":    "elastic-check",
				"severity": "critical",
			},
			Annotations: map[string]string{
				"source_type": "collectd",
				"domain":      "heartbeat",
			},
		},
		Summary: "collectd heartbeat elastic-check critical",
	},
	{
		Alert: PrometheusAlert{
			Labels: map[string]string{
				"interface": "lo",
				"service":   "collectd",
				"severity":  "FAILURE",
			},
			Annotations: map[string]string{
				"DataSource":  "rx",
				"source_type": "collectd",
				"summary": "Host localhost.localdomain, plugin interface (instance lo) type if_octets: " +
					"Data source \"rx\" is currently 43596.224329. That is above the failure threshold of 0.000000.",
			},
		},
		Summary: "Host localhost.localdomain, plugin interface (instance lo) type if_octets: " +
			"Data source \"rx\" is currently 43596.224329. That is above the failure threshold of 0.000000.",
	},
}

func TestAlert(t *testing.T) {
	t.Run("Alert summary generation.", func(t *testing.T) {
		for _, tCase := range alertTestCases {
			tCase.Alert.SetSummary()
			summary, ok := tCase.Alert.Annotations["summary"]
			require.True(t, ok)
			assert.Equal(t, tCase.Summary, summary)
		}
	})
}
