package collectd

import (
	"fmt"
	"math/rand"
	"time"
)

/*
{
	"values": [1270783],
	"dstypes": ["derive" ],
	"dsnames": ["value"],
	"time": 1515406948.326,
	"interval": 10.000,
	"host": "vtc126",
	"plugin": "cpu",
	"plugin_instance": "0",
	"type": "cpu",
	"type_instance": "user"
	}
*/

const cpuMetricTemplate = `{"values": [%d], "dstypes": ["derive"], "dsnames": ["value"],
                      "time": %f, "interval": %f, "host": "%s", "plugin": "cpu",
                      "plugin_instance": "%d","type": "cpu","type_instance": "user"}`

// GenCPUMetric ...
func GenCPUMetric(interval int, host string, count int) (mesg []byte) {
	msgBuffer := make([]byte, 0, 1024)

	msgBuffer = append(msgBuffer, "["...)

	for i := 0; i < count; i++ {
		msg := fmt.Sprintf(cpuMetricTemplate,
			rand.Int(), // val
			float64((time.Now().UnixNano()))/1000000000, // time
			10.0, // interval
			host, // host
			i)    // plugin_instance

		msgLen := len(msg)

		if len(msgBuffer)+msgLen+10 >= cap(msgBuffer) {
			fmt.Printf("Truncated to %d metrics / message\n", i)
			break
		}
		if i > 0 {
			msgBuffer = append(msgBuffer, ","...)
		}
		msgBuffer = append(msgBuffer, msg...)
	}
	msgBuffer = append(msgBuffer, "]"...)

	return msgBuffer
}
