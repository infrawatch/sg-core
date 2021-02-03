package main

import (
	"context"
	"time"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/transport"
)

const maxBufferSize = 4096

var msgBuffer []byte

var sent bool

var eventMessages = []string{
	// Ceilometer events
	`{"request":{"oslo.version":"2.0","oslo.message":` +
		`"{\"message_id\":\"4c9fbb58-c82d-4ca5-9f4c-2c61d0693214\",\"publisher_id\":\"telemetry.publisher.controller-0.redhat.local\",` +
		`\"event_type\":\"event\",\"priority\":\"SAMPLE\",\"payload\":[{\"message_id\":\"084c0bca-0d19-40c0-a724-9916e4815845\",` +
		`\"event_type\":\"image.delete\",\"generated\":\"2020-03-06T14:13:29.497096\",\"traits\":[[\"service\",1,\"image.localhost\"],` +
		`[\"project_id\",1,\"0f500647077b47f08a8ca9181e9b7aef\"],[\"user_id\",1,\"0f500647077b47f08a8ca9181e9b7aef\"],` +
		`[\"resource_id\",1,\"c4f7e00b-df85-4b77-9e1a-26a1de4d5735\"],[\"name\",1,\"cirros\"],[\"status\",1,\"deleted\"],` +
		`[\"created_at\",4,\"2020-03-06T14:01:07\"],[\"deleted_at\",4,\"2020-03-06T14:13:29\"],[\"size\",2,13287936]],\"raw\":{},` +
		`\"message_signature\":\"77e798b842991f9c0c35bda265fdf86075b4a1e58309db1d2adbf89386a3859e\"}],` +
		`\"timestamp\":\"2020-03-06 14:13:30.057411\"}"},"context": {}}}`,
	// Collectd events
	"[{\"labels\":{\"alertname\":\"collectd_connectivity_gauge\",\"instance\":\"d60b3c68f23e\",\"connectivity\":\"eno2\"," +
		"\"type\":\"interface_status\",\"severity\":\"FAILURE\",\"service\":\"collectd\"},\"annotations\":{\"summary\":\"\"," +
		"\"ves\":\"{\\\"domain\\\":\\\"stateChange\\\",\\\"eventId\\\":2,\\\"eventName\\\":\\\"interface eno2 up\\\"," +
		"\\\"lastEpochMicrosec\\\":1518790014024924,\\\"priority\\\":\\\"high\\\",\\\"reportingEntityName\\\":\\\"collectd connectivity plugin\\\"," +
		"\\\"sequence\\\":0,\\\"sourceName\\\":\\\"eno2\\\",\\\"startEpochMicrosec\\\":1518790009881440,\\\"version\\\":1.0," +
		"\\\"stateChangeFields\\\":{\\\"newState\\\":\\\"outOfService\\\",\\\"oldState\\\":\\\"inService\\\",\\\"stateChangeFieldsVersion\\\":1.0," +
		"\\\"stateInterface\\\":\\\"eno2\\\"}}\"},\"startsAt\":\"2018-02-16T14:06:54.024856417Z\"}]",
	"[{\"labels\":{\"alertname\":\"collectd_procevent_gauge\",\"instance\":\"d60b3c68f23e\",\"procevent\":\"bla.py\",\"type\":\"process_status\"," +
		"\"severity\":\"FAILURE\",\"service\":\"collectd\"},\"annotations\":{\"summary\":\"\",\"ves\":\"{\\\"domain\\\":\\\"fault\\\"," +
		"\\\"eventId\\\":3,\\\"eventName\\\":\\\"process bla.py (8537) down\\\",\\\"lastEpochMicrosec\\\":1518791119579620," +
		"\\\"priority\\\":\\\"high\\\",\\\"reportingEntityName\\\":\\\"collectd procevent plugin\\\",\\\"sequence\\\":0," +
		"\\\"sourceName\\\":\\\"bla.py\\\",\\\"startEpochMicrosec\\\":1518791111336973,\\\"version\\\":1.0,\\\"faultFields\\\":{" +
		"\\\"alarmCondition\\\":\\\"process bla.py (8537) state change\\\",\\\"alarmInterfaceA\\\":\\\"bla.py\\\"," +
		"\\\"eventSeverity\\\":\\\"CRITICAL\\\",\\\"eventSourceType\\\":\\\"process\\\",\\\"faultFieldsVersion\\\":1.0," +
		"\\\"specificProblem\\\":\\\"process bla.py (8537) down\\\",\\\"vfStatus\\\":\\\"Ready to terminate\\\"}}\"}," +
		"\"startsAt\":\"2018-02-16T14:25:19.579573212Z\"}]",
	`[{"labels":{"alertname":"collectd_interface_if_octets","instance":"localhost.localdomain","interface":"lo","severity":"FAILURE",` +
		`"service":"collectd"},"annotations":{"summary":"Host localhost.localdomain, plugin interface (instance lo) type if_octets: ` +
		`Data source \"rx\" is currently 43596.224329. That is above the failure threshold of 0.000000.","DataSource":"rx",` +
		`"CurrentValue":"43596.2243286703","WarningMin":"nan","WarningMax":"nan","FailureMin":"nan","FailureMax":"0"},` +
		`"startsAt":"2019-09-18T21:11:19.281603240Z"}]`,
	`[{"labels":{"alertname":"collectd_ovs_events_gauge","instance":"nfvha-comp-03","ovs_events":"br0","type":"link_status","severity":"OKAY",` +
		`"service":"collectd"},"annotations":{"summary":"link state of \"br0\" interface has been changed to \"UP\"",` +
		`"uuid":"c52f2aca-3cb1-48e3-bba3-100b54303d84"},"startsAt":"2018-02-22T20:12:19.547955618Z"}]`,
	`{"labels":{"check":"elastic-check","client":"wubba.lubba.dub.dub.redhat.com","severity":"FAILURE"},"annotations":` +
		`{"command":"podman ps | grep elastic || exit 2","duration":0.043278607,"executed":1601900769,"issued":1601900769,` +
		`"output":"time=\"2020-10-05T14:26:09+02:00\" level=error msg=\"cannot mkdir /run/user/0/libpod: mkdir /run/user/0/libpod: permission denied\"\n",` +
		`"status":2,"ves":"{\"commonEventHeader\":{\"domain\":\"heartbeat\",\"eventType\":\"checkResult\",` +
		`\"eventId\":\"wubba.lubba.dub.dub.redhat.com-elastic-check\",\"priority\":\"High\",\"reportingEntityId\":\"918e8d04-c5ae-4e20-a763-8eb4f1af7c80\",` +
		`\"reportingEntityName\":\"wubba.lubba.dub.dub.redhat.com\",\"sourceId\":\"918e8d04-c5ae-4e20-a763-8eb4f1af7c80\",` +
		`\"sourceName\":\"wubba.lubba.dub.dub.redhat.com-collectd-sensubility\",\"startingEpochMicrosec\":1601900769,\"lastEpochMicrosec\":1601900769},` +
		`\"heartbeatFields\":{\"additionalFields\":{\"check\":\"elastic-check\",\"command\":\"podman ps | grep elastic || exit 2 || $0\",` +
		`\"duration\":\"0.043279\",\"executed\":\"1601900769\",\"issued\":\"1601900769\",\"output\":\"time=\\\"2020-10-05T14:26:09+02:00\\\" ` +
		`level=error msg=\\\"cannot mkdir /run/user/0/libpod: mkdir /run/user/0/libpod: permission denied\\\"\\n\",\"status\":\"2\"}}}"},` +
		`"startsAt":"2020-10-05T14:26:09+02:00"}`,
}

func init() {
	msgBuffer = make([]byte, maxBufferSize)
}

//DummyEvents plugin struct
type DummyEvents struct {
}

//Run implements type Transport
func (de *DummyEvents) Run(ctx context.Context, wrFn transport.WriteFn, done chan bool) {

	for {
		for _, evt := range eventMessages {
			select {
			case <-ctx.Done():
				goto done
			case <-time.After(time.Second * 1):
				time.Sleep(time.Second * 1)
				wrFn([]byte(evt))
			}
		}
	}

done:
}

//Listen ...
func (de *DummyEvents) Listen(e data.Event) {

}

//Config load configurations
func (de *DummyEvents) Config(c []byte) error {
	return nil
}

//New create new socket transport
func New(l *logging.Logger) transport.Transport {
	return &DummyEvents{}
}
