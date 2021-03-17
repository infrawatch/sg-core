package ceilometer

import (
	jsontest "encoding/json"
	"testing"

	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/stretchr/testify/assert"
)

type parsingTestCase struct {
	EventBlob []byte
	RawMsg    rawMessage
	Parsed    Ceilometer
	Name      string
	Timestamp float64
	Traits    map[string]interface{}
	Event     data.Event
}

// NOTE: more test cases will come with coming sg-core usage and bug reports
var parsingCases = []parsingTestCase{
	{ // standard ceilometer event
		EventBlob: []byte(`{"request":{"oslo.version":"2.0","oslo.message":` +
			`"{\"message_id\":\"4c9fbb58-c82d-4ca5-9f4c-2c61d0693214\",\"publisher_id\":\"telemetry.publisher.controller-0.redhat.local\",` +
			`\"event_type\":\"event\",\"priority\":\"SAMPLE\",\"payload\":[{\"message_id\":\"084c0bca-0d19-40c0-a724-9916e4815845\",` +
			`\"event_type\":\"image.delete\",\"generated\":\"2020-03-06T14:13:29.497096\",\"traits\":[[\"service\",1,\"image.localhost\"],` +
			`[\"project_id\",1,\"0f500647077b47f08a8ca9181e9b7aef\"],[\"user_id\",1,\"0f500647077b47f08a8ca9181e9b7aef\"],` +
			`[\"resource_id\",1,\"c4f7e00b-df85-4b77-9e1a-26a1de4d5735\"],[\"name\",1,\"cirros\"],[\"status\",1,\"deleted\"],` +
			`[\"created_at\",4,\"2020-03-06T14:01:07\"],[\"deleted_at\",4,\"2020-03-06T14:13:29\"],[\"size\",2,13287936]],\"raw\":{},` +
			`\"message_signature\":\"77e798b842991f9c0c35bda265fdf86075b4a1e58309db1d2adbf89386a3859e\"}],` +
			`\"timestamp\":\"2020-03-06 14:13:30.057411\"}"},"context": {}}`),
		RawMsg: rawMessage{
			Request: osloRequest{
				OsloVersion: "2.0",
				OsloMessage: "{\"message_id\":\"4c9fbb58-c82d-4ca5-9f4c-2c61d0693214\",\"publisher_id\":\"telemetry.publisher.controller-0.redhat.local\"," +
					"\"event_type\":\"event\",\"priority\":\"SAMPLE\",\"payload\":[{\"message_id\":\"084c0bca-0d19-40c0-a724-9916e4815845\"," +
					"\"event_type\":\"image.delete\",\"generated\":\"2020-03-06T14:13:29.497096\",\"traits\":[[\"service\",1,\"image.localhost\"]," +
					"[\"project_id\",1,\"0f500647077b47f08a8ca9181e9b7aef\"],[\"user_id\",1,\"0f500647077b47f08a8ca9181e9b7aef\"]," +
					"[\"resource_id\",1,\"c4f7e00b-df85-4b77-9e1a-26a1de4d5735\"],[\"name\",1,\"cirros\"],[\"status\",1,\"deleted\"]," +
					"[\"created_at\",4,\"2020-03-06T14:01:07\"],[\"deleted_at\",4,\"2020-03-06T14:13:29\"],[\"size\",2,13287936]]," +
					"\"raw\":{},\"message_signature\":\"77e798b842991f9c0c35bda265fdf86075b4a1e58309db1d2adbf89386a3859e\"}]," +
					"\"timestamp\":\"2020-03-06 14:13:30.057411\"}",
			},
		},
		Parsed: Ceilometer{
			osloMessage: osloMessage{
				EventType:   "event",
				PublisherID: "telemetry.publisher.controller-0.redhat.local",
				Timestamp:   "2020-03-06 14:13:30.057411",
				Priority:    "SAMPLE",
				Payload: []osloPayload{
					{
						MessageID: "084c0bca-0d19-40c0-a724-9916e4815845",
						EventType: "image.delete",
						Generated: "2020-03-06T14:13:29.497096",
						Traits: []interface{}{
							[]interface{}{"service", float64(1), "image.localhost"},
							[]interface{}{"project_id", float64(1), "0f500647077b47f08a8ca9181e9b7aef"},
							[]interface{}{"user_id", float64(1), "0f500647077b47f08a8ca9181e9b7aef"},
							[]interface{}{"resource_id", float64(1), "c4f7e00b-df85-4b77-9e1a-26a1de4d5735"},
							[]interface{}{"name", float64(1), "cirros"},
							[]interface{}{"status", float64(1), "deleted"},
							[]interface{}{"created_at", float64(4), "2020-03-06T14:01:07"},
							[]interface{}{"deleted_at", float64(4), "2020-03-06T14:13:29"},
							[]interface{}{"size", float64(2), 1.3287936e+07},
						},
					},
				},
			},
		},
		Name:      "ceilometer_image",
		Timestamp: 1583504009,
		Traits: map[string]interface{}{
			"service":     "image.localhost",
			"project_id":  "0f500647077b47f08a8ca9181e9b7aef",
			"user_id":     "0f500647077b47f08a8ca9181e9b7aef",
			"resource_id": "c4f7e00b-df85-4b77-9e1a-26a1de4d5735",
			"name":        "cirros",
			"status":      "deleted",
			"created_at":  "2020-03-06T14:01:07",
			"deleted_at":  "2020-03-06T14:13:29",
			"size":        1.3287936e+07,
		},
		Event: data.Event{
			Index:     "ceilometer_image",
			Time:      1583504009,
			Type:      data.EVENT,
			Publisher: "telemetry.publisher.controller-0.redhat.local",
			Severity:  data.INFO,
			Labels: map[string]interface{}{
				"service":     "image.localhost",
				"project_id":  "0f500647077b47f08a8ca9181e9b7aef",
				"user_id":     "0f500647077b47f08a8ca9181e9b7aef",
				"resource_id": "c4f7e00b-df85-4b77-9e1a-26a1de4d5735",
				"name":        "cirros",
				"status":      "deleted",
				"created_at":  "2020-03-06T14:01:07",
				"deleted_at":  "2020-03-06T14:13:29",
				"size":        1.3287936e+07,
			},
			Annotations: map[string]interface{}{
				"source_type":  "ceilometer",
				"processed_by": "sg",
			},
		},
	},
	{ // used for testing non-standard values in event
		EventBlob: []byte(`{"request":{"oslo.version":"2.0","oslo.message":` +
			`"{\"message_id\":\"4c9fbb58-c82d-4ca5-9f4c-2c61d0693214\",\"publisher_id\":\"telemetry.publisher\",` +
			`\"event_type\":\"wubba\",\"priority\":\"SAMPLE\",\"payload\":[{\"message_id\":\"084c0bca-0d19-40c0-a724-9916e4815845\",` +
			`\"traits\":[[\"service\",1,\"image.localhost\"],` +
			`[\"project_id\",1,\"0f500647077b47f08a8ca9181e9b7aef\"],[\"user_id\",1,\"0f500647077b47f08a8ca9181e9b7aef\"],` +
			`[\"resource_id\",1,\"c4f7e00b-df85-4b77-9e1a-26a1de4d5735\"],[\"name\",1,\"cirros\"],[\"status\",1,\"deleted\"],` +
			`[\"created_at\",4,\"2020-03-06T14:01:07\"],[\"deleted_at\",4,\"2020-03-06T14:13:29\"],[\"size\",2,13287936]],\"raw\":{},` +
			`\"message_signature\":\"77e798b842991f9c0c35bda265fdf86075b4a1e58309db1d2adbf89386a3859e\"}],` +
			`\"timestamp\":\"2020-03-06 14:13:30.057411\"}"},"context": {}}`),
		RawMsg: rawMessage{
			Request: osloRequest{
				OsloVersion: "2.0",
				OsloMessage: "{\"message_id\":\"4c9fbb58-c82d-4ca5-9f4c-2c61d0693214\",\"publisher_id\":\"telemetry.publisher\"," +
					"\"event_type\":\"wubba\",\"priority\":\"SAMPLE\",\"payload\":[{\"message_id\":\"084c0bca-0d19-40c0-a724-9916e4815845\"," +
					"\"traits\":[[\"service\",1,\"image.localhost\"]," +
					"[\"project_id\",1,\"0f500647077b47f08a8ca9181e9b7aef\"],[\"user_id\",1,\"0f500647077b47f08a8ca9181e9b7aef\"]," +
					"[\"resource_id\",1,\"c4f7e00b-df85-4b77-9e1a-26a1de4d5735\"],[\"name\",1,\"cirros\"],[\"status\",1,\"deleted\"]," +
					"[\"created_at\",4,\"2020-03-06T14:01:07\"],[\"deleted_at\",4,\"2020-03-06T14:13:29\"],[\"size\",2,13287936]]," +
					"\"raw\":{},\"message_signature\":\"77e798b842991f9c0c35bda265fdf86075b4a1e58309db1d2adbf89386a3859e\"}]," +
					"\"timestamp\":\"2020-03-06 14:13:30.057411\"}",
			},
		},
		Parsed: Ceilometer{
			osloMessage: osloMessage{
				EventType:   "wubba",
				PublisherID: "telemetry.publisher",
				Timestamp:   "2020-03-06 14:13:30.057411",
				Priority:    "SAMPLE",
				Payload: []osloPayload{
					{
						MessageID: "084c0bca-0d19-40c0-a724-9916e4815845",
						Generated: "",
						Traits: []interface{}{
							[]interface{}{"service", float64(1), "image.localhost"},
							[]interface{}{"project_id", float64(1), "0f500647077b47f08a8ca9181e9b7aef"},
							[]interface{}{"user_id", float64(1), "0f500647077b47f08a8ca9181e9b7aef"},
							[]interface{}{"resource_id", float64(1), "c4f7e00b-df85-4b77-9e1a-26a1de4d5735"},
							[]interface{}{"name", float64(1), "cirros"},
							[]interface{}{"status", float64(1), "deleted"},
							[]interface{}{"created_at", float64(4), "2020-03-06T14:01:07"},
							[]interface{}{"deleted_at", float64(4), "2020-03-06T14:13:29"},
							[]interface{}{"size", float64(2), 1.3287936e+07},
						},
					},
				},
			},
		},
		Name:      "ceilometer_wubba",
		Timestamp: 1583504010,
		Traits: map[string]interface{}{
			"service":     "image.localhost",
			"project_id":  "0f500647077b47f08a8ca9181e9b7aef",
			"user_id":     "0f500647077b47f08a8ca9181e9b7aef",
			"resource_id": "c4f7e00b-df85-4b77-9e1a-26a1de4d5735",
			"name":        "cirros",
			"status":      "deleted",
			"created_at":  "2020-03-06T14:01:07",
			"deleted_at":  "2020-03-06T14:13:29",
			"size":        1.3287936e+07,
		},
		Event: data.Event{
			Index:     "ceilometer_wubba",
			Time:      1583504010,
			Type:      data.EVENT,
			Publisher: "telemetry.publisher",
			Severity:  data.INFO,
			Labels: map[string]interface{}{
				"service":     "image.localhost",
				"project_id":  "0f500647077b47f08a8ca9181e9b7aef",
				"user_id":     "0f500647077b47f08a8ca9181e9b7aef",
				"resource_id": "c4f7e00b-df85-4b77-9e1a-26a1de4d5735",
				"name":        "cirros",
				"status":      "deleted",
				"created_at":  "2020-03-06T14:01:07",
				"deleted_at":  "2020-03-06T14:13:29",
				"size":        1.3287936e+07,
			},
			Annotations: map[string]interface{}{
				"source_type":  "ceilometer",
				"processed_by": "sg",
			},
		},
	},
	{ // used for testing non-standard values in event too
		EventBlob: []byte(`{"request":{"oslo.version":"2.0","oslo.message":` +
			`"{\"message_id\":\"4c9fbb58-c82d-4ca5-9f4c-2c61d0693214\",\"publisher_id\":\"telemetry.publisher\",` +
			`\"priority\":\"SAMPLE\",\"payload\":[{\"message_id\":\"084c0bca-0d19-40c0-a724-9916e4815845\",` +
			`\"traits\":[[\"service\",1,\"image.localhost\"],` +
			`[\"project_id\",1,\"0f500647077b47f08a8ca9181e9b7aef\"],[\"user_id\",1,\"0f500647077b47f08a8ca9181e9b7aef\"],` +
			`[\"resource_id\",1,\"c4f7e00b-df85-4b77-9e1a-26a1de4d5735\"],[\"name\",1,\"cirros\"],[\"status\",1,\"deleted\"],` +
			`[\"created_at\",4,\"2020-03-06T14:01:07\"],[\"deleted_at\",4,\"2020-03-06T14:13:29\"],[\"size\",2,13287936]],\"raw\":{},` +
			`\"message_signature\":\"77e798b842991f9c0c35bda265fdf86075b4a1e58309db1d2adbf89386a3859e\"}]}"},"context": {}}`),
		RawMsg: rawMessage{
			Request: osloRequest{
				OsloVersion: "2.0",
				OsloMessage: "{\"message_id\":\"4c9fbb58-c82d-4ca5-9f4c-2c61d0693214\",\"publisher_id\":\"telemetry.publisher\"," +
					"\"priority\":\"SAMPLE\",\"payload\":[{\"message_id\":\"084c0bca-0d19-40c0-a724-9916e4815845\"," +
					"\"traits\":[[\"service\",1,\"image.localhost\"]," +
					"[\"project_id\",1,\"0f500647077b47f08a8ca9181e9b7aef\"],[\"user_id\",1,\"0f500647077b47f08a8ca9181e9b7aef\"]," +
					"[\"resource_id\",1,\"c4f7e00b-df85-4b77-9e1a-26a1de4d5735\"],[\"name\",1,\"cirros\"],[\"status\",1,\"deleted\"]," +
					"[\"created_at\",4,\"2020-03-06T14:01:07\"],[\"deleted_at\",4,\"2020-03-06T14:13:29\"],[\"size\",2,13287936]]," +
					"\"raw\":{},\"message_signature\":\"77e798b842991f9c0c35bda265fdf86075b4a1e58309db1d2adbf89386a3859e\"}]}",
			},
		},
		Parsed: Ceilometer{
			osloMessage: osloMessage{
				EventType:   "",
				PublisherID: "telemetry.publisher",
				Timestamp:   "2020-03-06 14:13:30.057411",
				Priority:    "SAMPLE",
				Payload: []osloPayload{
					{
						MessageID: "084c0bca-0d19-40c0-a724-9916e4815845",
						Generated: "2020-03-06T14:13:29.497096",
						Traits: []interface{}{
							[]interface{}{"service", float64(1), "image.localhost"},
							[]interface{}{"project_id", float64(1), "0f500647077b47f08a8ca9181e9b7aef"},
							[]interface{}{"user_id", float64(1), "0f500647077b47f08a8ca9181e9b7aef"},
							[]interface{}{"resource_id", float64(1), "c4f7e00b-df85-4b77-9e1a-26a1de4d5735"},
							[]interface{}{"name", float64(1), "cirros"},
							[]interface{}{"status", float64(1), "deleted"},
							[]interface{}{"created_at", float64(4), "2020-03-06T14:01:07"},
							[]interface{}{"deleted_at", float64(4), "2020-03-06T14:13:29"},
							[]interface{}{"size", float64(2), 1.3287936e+07},
						},
					},
				},
			},
		},
		Name:      "ceilometer_generic",
		Timestamp: 0,
		Traits: map[string]interface{}{
			"service":     "image.localhost",
			"project_id":  "0f500647077b47f08a8ca9181e9b7aef",
			"user_id":     "0f500647077b47f08a8ca9181e9b7aef",
			"resource_id": "c4f7e00b-df85-4b77-9e1a-26a1de4d5735",
			"name":        "cirros",
			"status":      "deleted",
			"created_at":  "2020-03-06T14:01:07",
			"deleted_at":  "2020-03-06T14:13:29",
			"size":        1.3287936e+07,
		},
		Event: data.Event{
			Index:     "ceilometer_generic",
			Time:      0,
			Type:      data.EVENT,
			Publisher: "telemetry.publisher",
			Severity:  data.INFO,
			Labels: map[string]interface{}{
				"service":     "image.localhost",
				"project_id":  "0f500647077b47f08a8ca9181e9b7aef",
				"user_id":     "0f500647077b47f08a8ca9181e9b7aef",
				"resource_id": "c4f7e00b-df85-4b77-9e1a-26a1de4d5735",
				"name":        "cirros",
				"status":      "deleted",
				"created_at":  "2020-03-06T14:01:07",
				"deleted_at":  "2020-03-06T14:13:29",
				"size":        1.3287936e+07,
			},
			Annotations: map[string]interface{}{
				"source_type":  "ceilometer",
				"processed_by": "sg",
			},
		},
	},
}

func TestCeilometerEvents(t *testing.T) {
	t.Run("Test correct event parsing.", func(t *testing.T) {
		for _, testCase := range parsingCases {
			// test sanitizing
			rm := rawMessage{}
			err := jsontest.Unmarshal(testCase.EventBlob, &rm)
			assert.NoError(t, err)
			rm.sanitizeMessage()
			assert.Equal(t, testCase.RawMsg, rm)
			// test parsing
			ceilo := Ceilometer{}
			err = ceilo.Parse(testCase.EventBlob)
			assert.NoError(t, err)
			// test name
			assert.Equal(t, testCase.Name, ceilo.name(0))
			// test traits
			traits, err := ceilo.traits(0)
			assert.NoError(t, err)
			assert.Equal(t, testCase.Traits, traits)
			// test timestamp
			assert.Equal(t, testCase.Timestamp, ceilo.getTimeAsEpoch(0))
			// test publishing
			expected := testCase.Event
			err = ceilo.PublishEvents(func(evt data.Event) {
				assert.Equal(t, expected, evt)
			})
			assert.NoError(t, err)
		}
	})

}
