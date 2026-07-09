package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Collector holds Prometheus instruments for the SNMP collector.
// MQTT and buffer metrics are registered now so Phase 2 can wire them
// without renaming series used by alerting dashboards.
type Collector struct {
	PollTotal          prometheus.Counter
	PollSuccessTotal   prometheus.Counter
	PollFailureTotal   *prometheus.CounterVec
	BufferDepth        prometheus.Gauge
	BufferEnqueueTotal prometheus.Counter
	BufferFlushTotal   prometheus.Counter
	MQTTConnected      prometheus.Gauge
	MQTTPublishTotal   prometheus.Counter
	MQTTPublishFailure prometheus.Counter
}

// New registers all collector metrics on the default Prometheus registerer.
func New() *Collector {
	c := &Collector{
		PollTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "collector_poll_total",
			Help: "Total number of device poll attempts",
		}),
		PollSuccessTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "collector_poll_success_total",
			Help: "Total number of successful device polls",
		}),
		PollFailureTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "collector_poll_failure_total",
			Help: "Total number of failed device polls",
		}, []string{"device_id", "error_class"}),
		BufferDepth: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "collector_buffer_depth",
			Help: "Current local buffer depth (0 for stdout publisher)",
		}),
		BufferEnqueueTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "collector_buffer_enqueue_total",
			Help: "Total events enqueued for publish",
		}),
		BufferFlushTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "collector_buffer_flush_total",
			Help: "Total publish flush batches",
		}),
		MQTTConnected: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "collector_mqtt_connected",
			Help: "Whether MQTT is connected (0 until Phase 2)",
		}),
		MQTTPublishTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "collector_mqtt_publish_total",
			Help: "Total successful MQTT publishes (Phase 2)",
		}),
		MQTTPublishFailure: promauto.NewCounter(prometheus.CounterOpts{
			Name: "collector_mqtt_publish_failure_total",
			Help: "Total failed MQTT publishes (Phase 2)",
		}),
	}
	c.BufferDepth.Set(0)
	c.MQTTConnected.Set(0)
	return c
}

// Handler returns an HTTP handler for /metrics.
func Handler() http.Handler {
	return promhttp.Handler()
}

// ErrorClassTimeout is used when an SNMP operation times out.
const ErrorClassTimeout = "timeout"

// ErrorClassSNMP is used for SNMP protocol / transport errors.
const ErrorClassSNMP = "snmp_error"

// ErrorClassParse is used when SNMP responses cannot be parsed.
const ErrorClassParse = "parse_error"
