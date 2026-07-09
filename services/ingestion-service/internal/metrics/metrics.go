package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Ingestion holds Prometheus instruments for the ingestion service.
type Ingestion struct {
	MessagesReceived     prometheus.Counter
	MessagesAccepted     prometheus.Counter
	MessagesRejected     prometheus.Counter
	MessagesDeduplicated prometheus.Counter
	DBWriteFailure       prometheus.Counter
	ProcessingDuration   prometheus.Histogram
	MQTTConnected        prometheus.Gauge
}

// New registers all ingestion metrics on the default Prometheus registerer.
func New() *Ingestion {
	return NewWithRegisterer(prometheus.DefaultRegisterer)
}

// NewWithRegisterer registers ingestion metrics on reg (useful in tests).
func NewWithRegisterer(reg prometheus.Registerer) *Ingestion {
	factory := prometheus.WrapRegistererWith(nil, reg)
	m := &Ingestion{
		MessagesReceived: mustCounter(factory, prometheus.CounterOpts{
			Name: "ingestion_messages_received_total",
			Help: "Total MQTT messages received",
		}),
		MessagesAccepted: mustCounter(factory, prometheus.CounterOpts{
			Name: "ingestion_messages_accepted_total",
			Help: "Total messages validated and persisted",
		}),
		MessagesRejected: mustCounter(factory, prometheus.CounterOpts{
			Name: "ingestion_messages_rejected_total",
			Help: "Total validation failures",
		}),
		MessagesDeduplicated: mustCounter(factory, prometheus.CounterOpts{
			Name: "ingestion_messages_deduplicated_total",
			Help: "Total duplicate messages skipped",
		}),
		DBWriteFailure: mustCounter(factory, prometheus.CounterOpts{
			Name: "ingestion_db_write_failure_total",
			Help: "Total database transaction failures",
		}),
		ProcessingDuration: mustHistogram(factory, prometheus.HistogramOpts{
			Name:    "ingestion_processing_duration_seconds",
			Help:    "End-to-end processing time per message (receive to ACK decision)",
			Buckets: prometheus.DefBuckets,
		}),
		MQTTConnected: mustGauge(factory, prometheus.GaugeOpts{
			Name: "ingestion_mqtt_connected",
			Help: "Whether MQTT is connected (1=connected, 0=disconnected)",
		}),
	}
	m.MQTTConnected.Set(0)
	return m
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

func mustGauge(reg prometheus.Registerer, opts prometheus.GaugeOpts) prometheus.Gauge {
	g := prometheus.NewGauge(opts)
	reg.MustRegister(g)
	return g
}

func mustHistogram(reg prometheus.Registerer, opts prometheus.HistogramOpts) prometheus.Histogram {
	h := prometheus.NewHistogram(opts)
	reg.MustRegister(h)
	return h
}
