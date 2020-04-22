package unixserver

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/infrawatch/sg2/pkg/cacheutil"
	"github.com/infrawatch/sg2/pkg/collectd"
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
		//***** There are metrics missing here:
		// collectd_last_pull_timestamp_seconds (Unused)
		// collectd_qpid_router_status (Used in perftest dashboard, but not that useful in practice, also hard to propagate via the bridge)
		// collectd_total_amqp_reconnect_count (Unused, same as above though)
		// collectd_elasticsearch_status (Unused, events specific so not for this codebase yet)
		// collectd_last_metric_for_host_status (Used in rhos-dashboard - could the be done a different way?)
		// collectd_metric_per_host (Unused)
		totalMetricsReceivedDesc: prometheus.NewDesc("sg_total_metric_rcv_count",
			"Total count of collectd metrics rcv'd.",
			nil, plabels,
		),
		totalAmqpReceivedDesc: prometheus.NewDesc("sg_total_amqp_rcv_count",
			"Total count of amqp msq rcv'd.",
			nil, plabels,
		),
		totalDecodeErrorsDesc: prometheus.NewDesc("sg_total_metric_decode_error_count",
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

	name = "collectd_" + cd.Plugin + "_" + cd.Type
	if cd.Type == cd.Plugin {
		name = "collectd_" + cd.Plugin
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

type deleteFn func()

type CDMetric struct {
	host           string
	pluginInstance string
	typeInstance   string
	metric         float64
	timeStamp      time.Time
	valueType      prometheus.ValueType
	metricDesc     *prometheus.Desc
	interval       float64

	lastArrival time.Time
	deleteFn    deleteFn
}

func (b *CDMetric) keepAlive() {
	b.lastArrival = time.Now()
}

func (b *CDMetric) staleTime() float64 {
	return time.Now().Sub(b.lastArrival).Seconds()
}

// Expired implements cacheutil.Expiry
func (b *CDMetric) Expired() bool {
	return (b.staleTime() >= b.interval)
}

// Delete implements cacheutil.Expiry
func (b *CDMetric) Delete() {
	b.deleteFn()
}

type CDMetrics struct {
	descriptions *CDMetricDescriptions
	// map[metricName]
	metrics map[string]map[string]*CDMetric
}

// NewCDMetrics  CDMetrics factory
func NewCDMetrics() (m *CDMetrics) {
	m = &CDMetrics{descriptions: NewCDMetricDescriptions(),
		// Indexed by metricName, then by "" + cd.Host + pluginInstance + typeInstance
		metrics: make(map[string]map[string]*CDMetric)}

	return
}

func (a *CDMetrics) updateOrAddMetric(cd *collectd.Collectd, index int, cs *cacheutil.CacheServer) error {

	if cd.Host == "" {
		return fmt.Errorf("missing host: %v ", cd)
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
		metric.timeStamp = cd.Time.Time()
		metric.keepAlive()
	} else {
		metric := &CDMetric{
			host:           cd.Host,
			pluginInstance: pluginInstance,
			typeInstance:   typeInstance,
			metric:         value,
			timeStamp:      cd.Time.Time(),
			metricDesc:     desc,
			valueType:      valueType,
			interval:       cd.Interval * 5,
		}
		metric.keepAlive()
		if a.metrics[metricName] == nil {
			a.metrics[metricName] = make(map[string]*CDMetric)
		}

		a.metrics[metricName][labelKey] = metric
		fmt.Printf("Add metric: %v\n", cd)

		metric.deleteFn = func() {
			fmt.Printf("Label %s in metric %s deleted after %fs of inactivity\n", labelKey, metricName, a.metrics[metricName][labelKey].staleTime())
			delete(a.metrics[metricName], labelKey)

			if len(a.metrics[metricName]) == 0 {
				delete(a.metrics, metricName)
				fmt.Printf("Metrics %s deleted\n", metricName)
			}
		}

		cs.Register(metric)
	}

	return nil
}

func (a *CDMetrics) updateOrAddMetrics(cdMetric *collectd.Collectd, cs *cacheutil.CacheServer) {
	for index := range cdMetric.Dsnames {
		err := a.updateOrAddMetric(cdMetric, index, cs)
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

//Collect implements prometheus.Collector
func (a *CDMetrics) Collect(ch chan<- prometheus.Metric) {
	for _, metric := range a.metrics {
		for _, labeledMetric := range metric {

			// get this from cache instead
			ch <- prometheus.NewMetricWithTimestamp(labeledMetric.timeStamp, prometheus.MustNewConstMetric(labeledMetric.metricDesc, labeledMetric.valueType, labeledMetric.metric,
				labeledMetric.host, labeledMetric.pluginInstance, labeledMetric.typeInstance))
		}
	}
}

//Listen ...
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

	// cache server
	cache := cacheutil.NewCacheServer()
	go cache.Run(ctx)

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
				allMetrics.updateOrAddMetrics(&m, cache)
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
