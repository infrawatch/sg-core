package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/application"
	"github.com/infrawatch/sg-core/pkg/concurrent"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type configT struct {
	Host          string
	Port          int `validate:"required"`
	MetricTimeout int
}

// used to expire stale metrics
type metricExpiry struct {
	sync.RWMutex
	lastArrival time.Time
	delete      func()
}

func (me *metricExpiry) keepAlive() {
	me.Lock()
	defer me.Unlock()
	me.lastArrival = time.Now()
}

func (me *metricExpiry) Expired(interval time.Duration) bool {
	me.RLock()
	defer me.RUnlock()
	return (time.Since(me.lastArrival) >= interval)
}

func (me *metricExpiry) Delete() {
	me.Lock()
	defer me.Unlock()
	me.delete()
}

type collectorExpiry struct {
	sync.RWMutex
	collector *PromCollector
	delete    func()
}

func (ce *collectorExpiry) Expired(interval time.Duration) bool {
	if ce.collector.descriptions.Len() == 0 {
		return true
	}
	return false
}

func (ce *collectorExpiry) Delete() {
	ce.Lock()
	defer ce.Unlock()
	ce.delete()
}

type logWrapper struct {
	l      *logging.Logger
	plugin string
}

func (lw *logWrapper) Error(msg string, err error) {
	lw.l.Metadata(logging.Metadata{"plugin": lw.plugin, "error": err})
	lw.l.Error(msg)
}

func (lw *logWrapper) Warn(msg string) {
	lw.l.Metadata(logging.Metadata{"plugin": lw.plugin})
	lw.l.Warn(msg)
}

func (lw *logWrapper) Infof(format string, a ...interface{}) {
	lw.l.Metadata(logging.Metadata{"plugin": lw.plugin})
	lw.l.Info(fmt.Sprintf(format, a...))
}

//PromCollector implements prometheus.Collector for incoming metrics. Metrics
// with differing label dimensions must create separate PromCollectors.
type PromCollector struct {
	logger       *logWrapper
	descriptions *concurrent.Map
	metrics      *concurrent.Map
	expirys      *concurrent.Map
	dimensions   int
}

//NewPromCollector PromCollector constructor
func NewPromCollector(l *logWrapper) *PromCollector {
	return &PromCollector{
		logger:       l,
		descriptions: concurrent.NewMap(),
		metrics:      concurrent.NewMap(),
		expirys:      concurrent.NewMap(),
	}
}

//Describe implements prometheus.Collector
func (pc *PromCollector) Describe(ch chan<- *prometheus.Desc) {
	for desc := range pc.descriptions.Iter() {
		ch <- desc.Value.(*prometheus.Desc)
	}
}

//Collect implements prometheus.Collector
func (pc *PromCollector) Collect(ch chan<- prometheus.Metric) {
	errs := []error{}
	for item := range pc.metrics.Iter() {

		metric := item.Value.(*data.Metric)
		desc := pc.descriptions.Get(metric.Name)
		pMetric, err := prometheus.NewConstMetric(desc.(*prometheus.Desc), metricTypeToPromValueType(metric.Type), metric.Value, metric.LabelVals...)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if !metric.Time.IsZero() {
			ch <- prometheus.NewMetricWithTimestamp(metric.Time, pMetric)
			continue
		}
		ch <- pMetric
	}

	for _, e := range errs {
		pc.logger.Error("prometheus failed scrapping metric", e)
	}
}

//Dimensions return dimension size of labels in collector
func (pc *PromCollector) Dimensions() int {
	return pc.dimensions
}

// SetDescs update prometheus descriptions
func (pc *PromCollector) SetDescs(name string, description string, labelKeys []string) error {
	if !pc.descriptions.Contains(name) {
		//fmt.Println("set desc")
		pc.descriptions.Set(name, prometheus.NewDesc(name, description, labelKeys, nil))
	}
	return nil
}

//UpdateMetrics update metrics in collector
func (pc *PromCollector) UpdateMetrics(name string, time time.Time, typ data.MetricType, interval time.Duration, value float64, labelKeys []string, labelVals []string, ep *expiryProc) {

	if !pc.expirys.Contains(name) { //register new metrics in expiry
		exp := &metricExpiry{
			delete: func() {
				pc.metrics.Delete(name)
				pc.descriptions.Delete(name)
				pc.expirys.Delete(name)
				pc.logger.Infof("metric '%s' deleted after %.1fs of stale time", name, interval.Seconds())
			},
		}
		pc.expirys.Set(name, exp)
		ep.register(exp)
		exp.keepAlive()

		pc.metrics.Set(name, &data.Metric{
			Name:      name,
			LabelKeys: labelKeys,
			LabelVals: labelVals,
			Time:      time,
			Type:      typ,
			Interval:  interval,
			Value:     value,
		})
		err := pc.SetDescs(name, "", labelKeys)
		if err != nil {
			pc.logger.Error("error setting prometheus collector descriptions", err)
			return
		}
	}

	m := pc.metrics.Get(name).(*data.Metric)
	m.Name = name
	m.LabelKeys = labelKeys
	m.LabelVals = labelVals
	m.Time = time
	m.Type = typ
	m.Value = value
	pc.expirys.Get(name).(*metricExpiry).keepAlive()
}

//Prometheus plugin for interfacing with Prometheus. Metrics with the same dimensions
// are included in the same collectors even if the labels are different
type Prometheus struct {
	configuration       configT
	logger              *logWrapper
	collectors          sync.Map //collectors mapped according to label dimensions
	metricExpiryProcs   sync.Map //stores expiry processes based for each metric interval
	collectorExpiryProc *expiryProc
	registry            *prometheus.Registry
	ctx                 context.Context
}

//New constructor
func New(l *logging.Logger) application.Application {
	return &Prometheus{
		configuration: configT{
			Host:          "127.0.0.1",
			Port:          3000,
			MetricTimeout: 20,
		},
		logger: &logWrapper{
			l:      l,
			plugin: "Prometheus",
		},
		collectors:          sync.Map{},
		metricExpiryProcs:   sync.Map{},
		collectorExpiryProc: newExpiryProc(time.Duration(10) * time.Second),
	}
}

//RecieveMetric callback function for recieving metric from the bus
func (p *Prometheus) RecieveMetric(name string, time time.Time, typ data.MetricType, interval time.Duration, value float64, labelKeys []string, labelVals []string) {

	var expProc *expiryProc
	labelLen := len(labelKeys)
	var promCol *PromCollector
	pc, found := p.collectors.Load(labelLen)
	if !found {
		promCol = NewPromCollector(p.logger)
		ce := &collectorExpiry{
			collector: promCol,
			delete: func() {
				p.logger.Warn("prometheus collector expired")
				p.registry.Unregister(promCol)

				p.collectors.Delete(len(labelKeys))
			},
		}

		p.collectorExpiryProc.register(ce)

		p.collectors.Store(len(labelKeys), promCol)
		p.registry.MustRegister(promCol)
	} else {
		promCol = pc.(*PromCollector)
	}

	ep, exists := p.metricExpiryProcs.Load(interval)
	if !exists {
		expProc = newExpiryProc(interval)
		p.metricExpiryProcs.Store(interval, expProc)
		p.logger.Infof("registered expiry process with interval %d", interval)
		go expProc.run(p.ctx)
	} else {
		expProc = ep.(*expiryProc)
	}

	promCol.UpdateMetrics(name, time, typ, interval, value, labelKeys, labelVals, expProc)
}

//Run run scrape endpoint
func (p *Prometheus) Run(ctx context.Context, done chan bool) {
	p.ctx = ctx
	p.registry = prometheus.NewRegistry()

	//Set up Metric Exporter
	handler := http.NewServeMux()
	handler.Handle("/metrics", promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{}))
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`<html>
                                <head><title>Prometheus Exporter</title></head>
                                <body>cacheutil
                                <h1>Prometheus Exporter</h1>
                                <p><a href='/metrics'>Metrics</a></p>
                                </body>
								</html>`))
		if err != nil {
			p.logger.Error("HTTP error", err)
		}
	})

	//run exporter for prometheus to scrape
	metricsURL := fmt.Sprintf("%s:%d", p.configuration.Host, p.configuration.Port)
	p.logger.Infof("metric server at : %s", metricsURL)

	srv := &http.Server{Addr: metricsURL}
	srv.Handler = handler

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			p.logger.Error("metric scrape endpoint failed", err)
			done <- true
		}
	}()

	//run collector expiry process
	go p.collectorExpiryProc.run(ctx)

	<-ctx.Done()
	timeout, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if err := srv.Shutdown(timeout); err != nil {
		p.logger.Error("error while shutting down metrics endpoint", err)
	}
	p.logger.Infof("exited")
}

//Config implements application.Application
func (p *Prometheus) Config(c []byte) error {
	p.configuration = configT{}
	err := config.ParseConfig(bytes.NewReader(c), &p.configuration)
	if err != nil {
		return err
	}
	return nil
}

// helper functions

func metricTypeToPromValueType(mType data.MetricType) prometheus.ValueType {
	return map[data.MetricType]prometheus.ValueType{
		data.COUNTER: prometheus.CounterValue,
		data.GAUGE:   prometheus.GaugeValue,
		data.UNTYPED: prometheus.UntypedValue,
	}[mType]
}
