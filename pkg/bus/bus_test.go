package bus

// var sampleMetrics []data.Metric = []data.Metric{
// 	{
// 		Name:  "collectd_metric_type0_samples_total",
// 		Value: 2121.0,
// 		Type:  data.COUNTER,
// 		Labels: map[string]string{
// 			"host":            "localhost",
// 			"plugin_instance": "base",
// 			"type_instance":   "base",
// 		},
// 	}, {
// 		Name:  "collectd_metric_type0_samples_total",
// 		Value: 2121.0,
// 		Type:  data.COUNTER,
// 		Labels: map[string]string{
// 			"host":            "localhost",
// 			"plugin_instance": "base",
// 			"type_instance":   "base",
// 		},
// 	}, {
// 		Name:  "collectd_metric_type0_samples_total",
// 		Value: 1010.0,
// 		Type:  data.COUNTER,
// 		Labels: map[string]string{
// 			"host":            "localhost",
// 			"plugin_instance": "base",
// 			"type_instance":   "base",
// 		},
// 	},
// 	{
// 		Name:  "collectd_metric_type0_samples_total",
// 		Value: 2121.0,
// 		Type:  data.COUNTER,
// 		Labels: map[string]string{
// 			"host":            "localhost",
// 			"plugin_instance": "base",
// 			"type_instance":   "base",
// 		},
// 	}, {
// 		Name:  "collectd_metric_type0_samples_total",
// 		Value: 2121.0,
// 		Type:  data.COUNTER,
// 		Labels: map[string]string{
// 			"host":            "localhost",
// 			"plugin_instance": "base",
// 			"type_instance":   "base",
// 		},
// 	}, {
// 		Name:  "collectd_metric_type0_samples_total",
// 		Value: 1010.0,
// 		Type:  data.COUNTER,
// 		Labels: map[string]string{
// 			"host":            "localhost",
// 			"plugin_instance": "base",
// 			"type_instance":   "base",
// 		},
// 	},
// 	{
// 		Name:  "collectd_metric_type0_samples_total",
// 		Value: 2121.0,
// 		Type:  data.COUNTER,
// 		Labels: map[string]string{
// 			"host":            "localhost",
// 			"plugin_instance": "base",
// 			"type_instance":   "base",
// 		},
// 	}, {
// 		Name:  "collectd_metric_type0_samples_total",
// 		Value: 2121.0,
// 		Type:  data.COUNTER,
// 		Labels: map[string]string{
// 			"host":            "localhost",
// 			"plugin_instance": "base",
// 			"type_instance":   "base",
// 		},
// 	}, {
// 		Name:  "collectd_metric_type0_samples_total",
// 		Value: 1010.0,
// 		Type:  data.COUNTER,
// 		Labels: map[string]string{
// 			"host":            "localhost",
// 			"plugin_instance": "base",
// 			"type_instance":   "base",
// 		},
// 	},
// }

// func BenchmarkBus(b *testing.B) {
// 	//This is similar to a real life confiuration of a metric bus in the sg-core
// 	//On my laptop, a 4 channel bus handles ~188k m/s with GOMAXPROCS = 8
// 	mBus := MetricBus{}

// 	var channels []chan []data.Metric
// 	for i := 0; i < 4; i++ {
// 		channels = append(channels, make(chan []data.Metric))
// 		mBus.Subscribe(channels[len(channels)-1])
// 		go func() {
// 			<-channels[len(channels)-1]
// 		}()
// 	}

// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		mBus.Publish(sampleMetrics)
// 	}
// }
