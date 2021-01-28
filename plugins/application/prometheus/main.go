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
	logger          *logWrapper
	descriptions    *concurrent.Map
	metrics         *concurrent.Map
	metricLabelKeys *concurrent.Map //used to insure labels are always reported to prometheus in the same order
	expirys         *concurrent.Map
	dimensions      int
}

//NewPromCollector PromCollector constructor
func NewPromCollector(l *logWrapper) *PromCollector {
	return &PromCollector{
		logger:          l,
		descriptions:    concurrent.NewMap(),
		metrics:         concurrent.NewMap(),
		metricLabelKeys: concurrent.NewMap(),
		expirys:         concurrent.NewMap(),
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

		metric := item.Value.(data.Metric)
		labelKeys := pc.metricLabelKeys.Get(metric.Name).([]string)
		labelValues := make([]string, 0, len(labelKeys))
		for _, l := range labelKeys { // TODO: optimize this
			labelValues = append(labelValues, metric.Labels[l])
		}
		desc := pc.descriptions.Get(metric.Name)
		pMetric, err := prometheus.NewConstMetric(desc.(*prometheus.Desc), metricTypeToPromValueType(metric.Type), metric.Value, labelValues...)
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
func (pc *PromCollector) SetDescs(name string, description string, labels map[string]string) error {
	if pc.dimensions != 0 && len(labels) != pc.dimensions {
		return fmt.Errorf("collector cannot accept metrics with %d labels, expects %d", len(labels), pc.dimensions)
	}
	if !pc.descriptions.Contains(name) {
		for k := range labels {
			keys := []string{}
			if pc.metricLabelKeys.Contains(name) {
				keys = pc.metricLabelKeys.Get(name).([]string)
			}
			pc.metricLabelKeys.Set(name, append(keys, k))
		}
		pc.descriptions.Set(name, prometheus.NewDesc(name, description, pc.metricLabelKeys.Get(name).([]string), nil))
	}
	return nil
}

//UpdateMetrics update metrics in collector
func (pc *PromCollector) UpdateMetrics(metric data.Metric, ep *expiryProc) {
	if !pc.expirys.Contains(metric.Name) { //register new metrics in expiry
		exp := metricExpiry{
			delete: func() {
				pc.metrics.Delete(metric.Name)
				pc.descriptions.Delete(metric.Name)
				pc.expirys.Delete(metric.Name)
				pc.logger.Infof("metric '%s' deleted after %.1fs of stale time", metric.Name, metric.Interval.Seconds())
			},
		}
		pc.expirys.Set(metric.Name, &exp)
		ep.register(&exp)
	}
	pc.metrics.Set(metric.Name, metric)
	pc.expirys.Get(metric.Name).(*metricExpiry).keepAlive()
}

//Prometheus plugin for interfacing with Prometheus. Metrics with the same dimensions
// are included in the same collectors even if the labels are different
type Prometheus struct {
	configuration       configT
	logger              *logWrapper
	collectors          *concurrent.Map //collectors mapped according to label dimensions
	metricExpiryProcs   sync.Map        //stores expiry processes based for each metric interval
	collectorExpiryProc *expiryProc
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
		collectors:          concurrent.NewMap(),
		metricExpiryProcs:   sync.Map{},
		collectorExpiryProc: newExpiryProc(time.Duration(10) * time.Second),
	}
}

//Run run scrape endpoint
func (p *Prometheus) Run(ctx context.Context, eChan chan data.Event, mChan chan []data.Metric, done chan bool) {
	registry := prometheus.NewRegistry()

	//Set up Metric Exporter
	handler := http.NewServeMux()
	handler.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
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

	for {
		select {
		case <-ctx.Done():
			goto done
		case <-eChan:
			p.logger.Warn("event received - disregarding")
		case metrics := <-mChan:
			var expProc *expiryProc
			for _, m := range metrics {
				labelLenStr := fmt.Sprintf("%d", len(m.Labels))
				if !p.collectors.Contains(labelLenStr) {
					c := NewPromCollector(p.logger)
					ce := &collectorExpiry{
						collector: c,
						delete: func() {
							p.logger.Warn("prometheus collector expired")
							registry.Unregister(c)
							p.collectors.Delete(string(labelLenStr))
						},
					}

					p.collectorExpiryProc.register(ce)

					p.collectors.Set(string(labelLenStr), c)
					registry.MustRegister(c)
				}
				err := p.collectors.Get(labelLenStr).(*PromCollector).SetDescs(m.Name, "", m.Labels)
				if err != nil {
					p.logger.Error("error setting prometheus collector descriptions", err)
					continue
				}

				ep, exists := p.metricExpiryProcs.Load(m.Interval)
				if !exists {
					expProc = newExpiryProc(m.Interval)
					p.metricExpiryProcs.Store(m.Interval, expProc)
					p.logger.Infof("registered expiry process with interval %d", m.Interval)
					go expProc.run(ctx)
				} else {
					expProc = ep.(*expiryProc)
				}
				p.collectors.Get(labelLenStr).(*PromCollector).UpdateMetrics(m, expProc)
			}
		}
	}
done:
	timeout, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if err := srv.Shutdown(timeout); err != nil {
		p.logger.Error("error while shutting down metrics endpoint", err)
	}
	wg.Wait()
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
