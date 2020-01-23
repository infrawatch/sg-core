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

//AMQPHandler ...
type PromIntf struct {
	totalReceived         uint64
	totalDecodeErrors     uint64
	totalReceivedDesc     *prometheus.Desc
	totalDecodeErrorsDesc *prometheus.Desc
}

//NewAMQPHandler  ...
func NewPromIntf(source string) *PromIntf {
	plabels := prometheus.Labels{}
	plabels["source"] = source
	return &PromIntf{
		totalReceived:     0,
		totalDecodeErrors: 0,
		totalReceivedDesc: prometheus.NewDesc("cd_total_metric_rcv_count",
			"Total count of collectd metrics rcv'd.",
			nil, plabels,
		),
		totalDecodeErrorsDesc: prometheus.NewDesc("cd_total_metric_decode_error_count",
			"Total count of amqp message processed.",
			nil, plabels,
		),
	}
}

//IncTotalReceived ...
func (a *PromIntf) IncTotalReceived() {
	a.totalReceived++
}

//AddTotalReceived ...
func (a *PromIntf) AddTotalReceived(num int) {
	a.totalReceived += uint64(num)
}

//GetTotalReceived ...
func (a *PromIntf) GetTotalReceived() uint64 {
	return a.totalReceived
}

//IncTotalDecodeErrors ...
func (a *PromIntf) IncTotalDecodeErrors() {
	a.totalDecodeErrors++
}

//GetTotalDecodeErrors ...
func (a *PromIntf) GetTotalDecodeErrors() uint64 {
	return a.totalDecodeErrors
}

//Describe ...
func (a *PromIntf) Describe(ch chan<- *prometheus.Desc) {
	ch <- a.totalReceivedDesc
	ch <- a.totalDecodeErrorsDesc
}

//Collect implements prometheus.Collector.
func (a *PromIntf) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(a.totalReceivedDesc, prometheus.CounterValue, float64(a.totalReceived))
	ch <- prometheus.MustNewConstMetric(a.totalDecodeErrorsDesc, prometheus.CounterValue, float64(a.totalDecodeErrors))
}

const maxBufferSize = 4096

var msgBuffer []byte

func init() {
	msgBuffer = make([]byte, maxBufferSize)
}

func Listen(ctx context.Context, address string, w *bufio.Writer, registry *prometheus.Registry) (err error) {
	var laddr net.UnixAddr

	laddr.Name = address
	laddr.Net = "unixgram"

	os.Remove(address)

	pc, err := net.ListenUnixgram("unixgram", &laddr)
	if err != nil {

		return
	}
	defer os.Remove(address)

	promIntfMetrics := NewPromIntf("SG")

	registry.MustRegister(promIntfMetrics)

	myAddr := pc.LocalAddr()
	fmt.Printf("Listening on %s\n", myAddr)

	doneChan := make(chan error, 1)

	go func() {
		cd := new(collectd.Collectd)

		for {
			n, err := pc.Read(msgBuffer[:])
			if err != nil || n < 1 {
				doneChan <- err
				return
			}

			if w != nil {
				if _, err := w.WriteString(string(append(msgBuffer[:n], "\n"...))); err != nil {
					panic(err)
				}
			}

			metric, err := cd.ParseInputByte(msgBuffer)
			if err != nil {
				promIntfMetrics.IncTotalDecodeErrors()
			} else if (*metric)[0].Interval < 0.0 {
				doneChan <- err
			}

			promIntfMetrics.AddTotalReceived(len(*metric))
		}
	}()

	var lastCount uint64

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
			fmt.Printf("Rcv'd: %d(%d)\n", promIntfMetrics.GetTotalReceived(), promIntfMetrics.GetTotalReceived()-lastCount)
			lastCount = promIntfMetrics.GetTotalReceived()
		}
	}
done:
	return err
}
