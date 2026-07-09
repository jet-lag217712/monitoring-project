package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Collector holds Prometheus instruments for the SNMP collector.
type Collector struct {
	BufferDepth                prometheus.Gauge
	BufferEnqueueTotal         prometheus.Counter
	BufferFlushBatchesTotal    prometheus.Counter
	BufferFlushedMessagesTotal prometheus.Counter
	MQTTConnected              prometheus.Gauge
	MQTTPublishTotal           prometheus.Counter
	MQTTPublishFailure         prometheus.Counter
	PollTotal                  prometheus.Counter
	PollSuccessTotal           prometheus.Counter
	PollFailureTotal           *prometheus.CounterVec
}

// New registers all collector metrics on the default Prometheus registerer.
func New() *Collector {
	return NewWithRegisterer(prometheus.DefaultRegisterer)
}

// NewWithRegisterer registers collector metrics on reg (useful in tests).
func NewWithRegisterer(reg prometheus.Registerer) *Collector {
	factory := prometheus.WrapRegistererWith(nil, reg)
	c := &Collector{
		PollTotal: mustCounter(factory, prometheus.CounterOpts{
			Name: "collector_poll_total",
			Help: "Total number of device poll attempts",
		}),
		PollSuccessTotal: mustCounter(factory, prometheus.CounterOpts{
			Name: "collector_poll_success_total",
			Help: "Total number of successful device polls",
		}),
		PollFailureTotal: mustCounterVec(factory, prometheus.CounterOpts{
			Name: "collector_poll_failure_total",
			Help: "Total number of failed device polls",
		}, []string{"device_id", "error_class"}),
		BufferDepth: mustGauge(factory, prometheus.GaugeOpts{
			Name: "collector_buffer_depth",
			Help: "Current local buffer depth",
		}),
		BufferEnqueueTotal: mustCounter(factory, prometheus.CounterOpts{
			Name: "collector_buffer_enqueue_total",
			Help: "Total events enqueued for publish",
		}),
		BufferFlushBatchesTotal: mustCounter(factory, prometheus.CounterOpts{
			Name: "collector_buffer_flush_batches_total",
			Help: "Total flush batches that published at least one message",
		}),
		BufferFlushedMessagesTotal: mustCounter(factory, prometheus.CounterOpts{
			Name: "collector_buffer_flushed_messages_total",
			Help: "Total messages successfully flushed from the buffer",
		}),
		MQTTConnected: mustGauge(factory, prometheus.GaugeOpts{
			Name: "collector_mqtt_connected",
			Help: "Whether MQTT is connected (1=connected, 0=disconnected)",
		}),
		MQTTPublishTotal: mustCounter(factory, prometheus.CounterOpts{
			Name: "collector_mqtt_publish_total",
			Help: "Total successful MQTT publishes",
		}),
		MQTTPublishFailure: mustCounter(factory, prometheus.CounterOpts{
			Name: "collector_mqtt_publish_failure_total",
			Help: "Total failed MQTT publishes",
		}),
	}
	c.BufferDepth.Set(0)
	c.MQTTConnected.Set(0)
	return c
}

// SetBufferDepth updates the Prometheus gauge from the in-memory depth counter.
func (c *Collector) SetBufferDepth(depth int64) {
	c.BufferDepth.Set(float64(depth))
}

// Handler returns an HTTP handler for /metrics.
func Handler() http.Handler {
	return promhttp.Handler()
}

func mustCounter(reg prometheus.Registerer, opts prometheus.CounterOpts) prometheus.Counter {
	c := prometheus.NewCounter(opts)
	reg.MustRegister(c)
	return c
}

func mustCounterVec(reg prometheus.Registerer, opts prometheus.CounterOpts, labels []string) *prometheus.CounterVec {
	c := prometheus.NewCounterVec(opts, labels)
	reg.MustRegister(c)
	return c
}

func mustGauge(reg prometheus.Registerer, opts prometheus.GaugeOpts) prometheus.Gauge {
	g := prometheus.NewGauge(opts)
	reg.MustRegister(g)
	return g
}

// ErrorClassTimeout is used when an SNMP operation times out.
const ErrorClassTimeout = "timeout"

// ErrorClassSNMP is used for SNMP protocol / transport errors.
const ErrorClassSNMP = "snmp_error"

// ErrorClassParse is used when SNMP responses cannot be parsed.
const ErrorClassParse = "parse_error"
