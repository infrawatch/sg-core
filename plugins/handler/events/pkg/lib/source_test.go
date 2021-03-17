package lib

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type dataSourceTestCase struct {
	EventBlob []byte
	Source    string
}

var dsCases = []dataSourceTestCase{
	dataSourceTestCase{
		Source: "ceilometer",
		EventBlob: []byte(`{"request":{"oslo.version":"2.0","oslo.message":` +
			`"{\"message_id\":\"4c9fbb58-c82d-4ca5-9f4c-2c61d0693214\",\"publisher_id\":\"telemetry.publisher\",` +
			`\"event_type\":\"wubba\",\"priority\":\"SAMPLE\",\"payload\":[{\"message_id\":\"084c0bca-0d19-40c0-a724-9916e4815845\",` +
			`\"traits\":[[\"service\",1,\"image.localhost\"],` +
			`[\"project_id\",1,\"0f500647077b47f08a8ca9181e9b7aef\"],[\"user_id\",1,\"0f500647077b47f08a8ca9181e9b7aef\"],` +
			`[\"resource_id\",1,\"c4f7e00b-df85-4b77-9e1a-26a1de4d5735\"],[\"name\",1,\"cirros\"],[\"status\",1,\"deleted\"],` +
			`[\"created_at\",4,\"2020-03-06T14:01:07\"],[\"deleted_at\",4,\"2020-03-06T14:13:29\"],[\"size\",2,13287936]],\"raw\":{},` +
			`\"message_signature\":\"77e798b842991f9c0c35bda265fdf86075b4a1e58309db1d2adbf89386a3859e\"}],` +
			`\"timestamp\":\"2020-03-06 14:13:30.057411\"}"},"context": {}}`),
	},
	dataSourceTestCase{
		Source: "collectd",
		EventBlob: []byte(`[{"labels":{"alertname":"collectd_interface_if_octets","instance":"localhost.localdomain","interface":"lo","severity":"FAILURE",` +
			`"service":"collectd"},"annotations":{"summary":"Host localhost.localdomain, plugin interface (instance lo) type if_octets: ` +
			`Data source \"rx\" is currently 43596.224329. That is above the failure threshold of 0.000000.","DataSource":"rx",` +
			`"CurrentValue":"43596.2243286703","WarningMin":"nan","WarningMax":"nan","FailureMin":"nan","FailureMax":"0"},` +
			`"startsAt":"2019-09-18T21:11:19.281603240Z"}]`),
	},
}

func TestDataSource(t *testing.T) {
	t.Run("Test correct data source recognizing.", func(t *testing.T) {
		for _, testCase := range dsCases {
			ds := DataSource(666)
			ds.SetFromMessage(testCase.EventBlob)
			assert.Equal(t, testCase.Source, ds.String())
		}
	})

}
