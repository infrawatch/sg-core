package unixserver

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/atyronesmith/sa-benchmark/pkg/collectd"
	"github.com/prometheus/client_golang/prometheus"
)

const maxBufferSize = 4096

var msgBuffer []byte

var (
	msgRecvd = prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "msg_rcv_total",
		Help:        "Number of json messages received.",
		ConstLabels: prometheus.Labels{"version": "1234"},
	})
)

func init() {
	msgBuffer = make([]byte, maxBufferSize)
}

func Listen(ctx context.Context, address string, w *bufio.Writer) (err error) {
	prometheus.MustRegister(msgRecvd)

	var laddr net.UnixAddr

	laddr.Name = address
	laddr.Net = "unixgram"

	os.Remove(address)

	pc, err := net.ListenUnixgram("unixgram", &laddr)
	if err != nil {

		return
	}
	defer os.Remove(address)

	myAddr := pc.LocalAddr()
	fmt.Printf("Listening on %s\n", myAddr)

	doneChan := make(chan error, 1)

	count := 0

	go func() {
		cd := new(collectd.Collectd)

		for {
			n, err := pc.Read(msgBuffer[:])
			if err != nil || n < 1 {
				doneChan <- err
				return
			}
			msgRecvd.Inc()

			if w != nil {
				if _, err := w.WriteString(string(append(msgBuffer[:n], "\n"...))); err != nil {
					panic(err)
				}
			}

			metric, err := cd.ParseInputByte(msgBuffer)
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
