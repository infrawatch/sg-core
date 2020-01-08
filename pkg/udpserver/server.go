package udpserver

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/atyronesmith/sa-benchmark/pkg/collectd"
	"github.com/prometheus/client_golang/prometheus"
)

const maxBufferSize = 1024

var (
	msgRecvd = prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "msg_rcv_total",
		Help:        "Number of json messages received.",
		ConstLabels: prometheus.Labels{"version": "1234"},
	})
)

func init() {
	prometheus.MustRegister(msgRecvd)
}

func Listen(ctx context.Context, address string) (err error) {

	pc, err := net.ListenPacket("udp", address)
	if err != nil {
		return
	}

	myAddr := pc.LocalAddr()
	fmt.Printf("Listening on %s\n", myAddr)

	defer pc.Close()

	doneChan := make(chan error, 1)
	buffer := make([]byte, maxBufferSize)

	count := 0

	go func() {
		cd := new(collectd.Collectd)

		for {
			n, _, err := pc.ReadFrom(buffer)
			if err != nil || n < 1 {
				doneChan <- err
				return
			}
			msgRecvd.Inc()
			metric, err := cd.ParseInputByte(buffer)
			if err != nil {
				fmt.Printf("Error parsing JSON!\n")
				doneChan <- err
			} else if (*metric)[0].Interval < 0.0 {
				doneChan <- err
			}
			count += len(*metric)
		}
	}()

	lastCount := 0
	for {
		select {
		case <-ctx.Done():
			fmt.Println("cancelled")
			err = ctx.Err()
			goto done
		case err = <-doneChan:
			goto done
		default:
			time.Sleep(time.Second * 1)
			fmt.Printf("Rcv'd: %d(%d)\n", count, count-lastCount)
			lastCount = count
		}
	}
done:
	return err
}
