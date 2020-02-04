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
	totalMetricsReceived     uint64
	totalAmqpReceived        uint64
	totalDecodeErrors        uint64
	totalMetricsReceivedDesc *prometheus.Desc
	totalAmqpReceivedDesc    *prometheus.Desc
	totalDecodeErrorsDesc    *prometheus.Desc
}

//NewAMQPHandler  ...
func NewPromIntf(source string) *PromIntf {
	plabels := prometheus.Labels{}
	plabels["source"] = source
	return &PromIntf{
		totalMetricsReceived: 0,
		totalDecodeErrors:    0,
		totalAmqpReceived:    0,
		totalMetricsReceivedDesc: prometheus.NewDesc("sa_total_metric_rcv_count",
			"Total count of collectd metrics rcv'd.",
			nil, plabels,
		),
		totalAmqpReceivedDesc: prometheus.NewDesc("sa_total_amqp_rcv_count",
			"Total count of amqp msq rcv'd.",
			nil, plabels,
		),
		totalDecodeErrorsDesc: prometheus.NewDesc("sa_total_metric_decode_error_count",
			"Total count of amqp message processed.",
			nil, plabels,
		),
	}
}

//IncTotalMetricsReceived ...
func (a *PromIntf) IncTotalMetricsReceived() {
	a.totalMetricsReceived++
}

//IncTtoalAmqpReceived ...
func (a *PromIntf) IncTotalAmqpReceived() {
	a.totalAmqpReceived++
}

//AddTotalReceived ...
func (a *PromIntf) AddTotalReceived(num int) {
	a.totalMetricsReceived += uint64(num)
}

//GetTotalReceived ...
func (a *PromIntf) GetTotalMetricsReceived() uint64 {
	return a.totalMetricsReceived
}

//GetTotalReceived ...
func (a *PromIntf) GetTotalAmqpReceived() uint64 {
	return a.totalAmqpReceived
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
	ch <- a.totalMetricsReceivedDesc
	ch <- a.totalAmqpReceivedDesc
	ch <- a.totalDecodeErrorsDesc
}

//Collect implements prometheus.Collector.
func (a *PromIntf) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(a.totalMetricsReceivedDesc, prometheus.CounterValue, float64(a.totalMetricsReceived))
	ch <- prometheus.MustNewConstMetric(a.totalAmqpReceivedDesc, prometheus.CounterValue, float64(a.totalAmqpReceived))
	ch <- prometheus.MustNewConstMetric(a.totalDecodeErrorsDesc, prometheus.CounterValue, float64(a.totalDecodeErrors))
}

const maxBufferSize = 4096

var msgBuffer []byte

func init() {
	msgBuffer = make([]byte, maxBufferSize)
}

func genMetricName(cd *collectd.Collectd, index int) (name string) {

	name = "cd_" + cd.Plugin + "_" + cd.Type
	if cd.Type == cd.Plugin {
		name = "cd_" + cd.Plugin
	}

	if dsname := cd.Dsnames[index]; dsname != "value" {
		name += "_" + dsname
	}

	switch cd.Dstypes[index] {
	case "counter", "derive":
		name += "_total"
	}

	return
}

type CDMetricDescription struct {
	metricName string
	metricDesc *prometheus.Desc
}

type CDMetricDescriptions struct {
	descriptions map[string]*CDMetricDescription
}

func NewCDMetricDescriptions() (metricDescriptions *CDMetricDescriptions) {
	metricDescriptions = &CDMetricDescriptions{make(map[string]*CDMetricDescription)}

	return
}

func (a *CDMetricDescriptions) getOrAddMetricDescription(cd *collectd.Collectd, metricName string) (desc *prometheus.Desc) {
	var found bool

	var metricDescription *CDMetricDescription

	if metricDescription, found = a.descriptions[metricName]; !found {
		metricDescription = &CDMetricDescription{metricName, prometheus.NewDesc(metricName,
			"", []string{"host", "plugin_instance", "type_instance"}, nil,
		)}
		a.descriptions[metricName] = metricDescription
	}

	desc = metricDescription.metricDesc

	return
}

type CDMetric struct {
	host           string
	pluginInstance string
	typeInstance   string
	metric         float64
	valueType      prometheus.ValueType
	metricDesc     *prometheus.Desc
}

type CDMetrics struct {
	descriptions *CDMetricDescriptions
	// map[metricName]
	metrics map[string]map[string]*CDMetric
}

func NewCDMetrics() (m *CDMetrics) {
	m = &CDMetrics{descriptions: NewCDMetricDescriptions(),
		metrics: make(map[string]map[string]*CDMetric)}

	return
}

func (a *CDMetrics) updateOrAddMetric(cd *collectd.Collectd, index int) error {

	if cd.Host == "" {
		return fmt.Errorf("Missing host: %v !", cd)
	}

	pluginInstance := cd.PluginInstance
	if pluginInstance == "" {
		pluginInstance = "base"
	}
	typeInstance := cd.TypeInstance
	if typeInstance == "" {
		typeInstance = "base"
	}
	// Keys are always in order, {host, plugin_instance, type_instance}
	// Concatenate and just use as hash?
	metricName := genMetricName(cd, index)

	desc := a.descriptions.getOrAddMetricDescription(cd, metricName)

	value := float64(cd.Values[index])

	// Convert to getOrAddMetric!

	var valueType prometheus.ValueType
	switch cd.Dstypes[index] {
	case "gauge":
		valueType = prometheus.GaugeValue
	case "counter", "derive":
		valueType = prometheus.CounterValue
	default:
		return fmt.Errorf("unknown name of value type: %s", cd.Dstypes[index])
	}
	labelKey := cd.Host + pluginInstance + typeInstance
	if metric, found := a.metrics[metricName][labelKey]; found {
		metric.metric = value
	} else {
		metric := &CDMetric{
			host:           cd.Host,
			pluginInstance: pluginInstance,
			typeInstance:   typeInstance,
			metric:         value,
			metricDesc:     desc,
			valueType:      valueType,
		}
		if a.metrics[metricName] == nil {
			a.metrics[metricName] = make(map[string]*CDMetric)
		}
		a.metrics[metricName][labelKey] = metric
		fmt.Printf("Add metric: %v\n", metric)
	}

	return nil
}

func (a *CDMetrics) updateOrAddMetrics(cdMetric *collectd.Collectd) {
	for index := range cdMetric.Dsnames {
		err := a.updateOrAddMetric(cdMetric, index)
		if err != nil {
			fmt.Printf("%+v\n", err)
		}
	}
}

//Describe ...
func (a *CDMetrics) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range a.descriptions.descriptions {
		ch <- desc.metricDesc
	}
}

//Collect implements prometheus.Collector.
func (a *CDMetrics) Collect(ch chan<- prometheus.Metric) {
	for _, metric := range a.metrics {
		for _, labeled_metric := range metric {
			ch <- prometheus.MustNewConstMetric(labeled_metric.metricDesc, labeled_metric.valueType, labeled_metric.metric,
				labeled_metric.host, labeled_metric.pluginInstance, labeled_metric.typeInstance)
		}
	}
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

	allMetrics := NewCDMetrics()

	registry.MustRegister(allMetrics)

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
			promIntfMetrics.IncTotalAmqpReceived()

			metrics, err := cd.ParseInputByte(msgBuffer)
			if err != nil {
				promIntfMetrics.IncTotalDecodeErrors()
				fmt.Printf("dd\n")
			} else if (*metrics)[0].Interval < 0.0 {
				doneChan <- err
			}
			promIntfMetrics.AddTotalReceived(len(*metrics))

			for _, m := range *metrics {
				allMetrics.updateOrAddMetrics(&m)
			}
		}
	}()

	var lastMetricCount, lastAmqpCount uint64

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
			fmt.Printf("Rcv'd: %d(%d) metrics, %d(%d) msgs\n", promIntfMetrics.GetTotalMetricsReceived(), promIntfMetrics.GetTotalMetricsReceived()-lastMetricCount,
				promIntfMetrics.GetTotalAmqpReceived(), promIntfMetrics.GetTotalAmqpReceived()-lastAmqpCount)
			lastMetricCount = promIntfMetrics.GetTotalMetricsReceived()
			lastAmqpCount = promIntfMetrics.GetTotalAmqpReceived()

		}
	}
done:
	return err
}
