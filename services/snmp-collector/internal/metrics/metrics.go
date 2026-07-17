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
	PollDuration               prometheus.Histogram
	ProfileDetectionTotal      *prometheus.CounterVec
	ProfileFallbackTotal       prometheus.Counter
	ProfileCollectionFailure   *prometheus.CounterVec
	ProfileDuration            prometheus.Histogram
	InterfaceSelectionTotal    *prometheus.CounterVec
	DiscoveryAttemptsTotal     prometheus.Counter
	DiscoveryCandidatesTotal   prometheus.Counter
	DiscoveryErrorsTotal       prometheus.Counter
	DiscoveryRateLimitWaits    prometheus.Counter
	ConfigReloadSuccessTotal   prometheus.Counter
	ConfigReloadFailureTotal   prometheus.Counter
	HealthTransitionsTotal     *prometheus.CounterVec
	HealthDevices              *prometheus.GaugeVec
	DependencyImpactedDevices  prometheus.Gauge
	HealthPendingFailures      prometheus.Gauge
	Ready                      prometheus.Gauge
	HeartbeatPublishTotal      prometheus.Counter
	HeartbeatPublishFailure    prometheus.Counter
	HeartbeatDuration          prometheus.Histogram
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
		PollDuration: mustHistogram(factory, prometheus.HistogramOpts{
			Name:    "collector_poll_duration_seconds",
			Help:    "Device poll pipeline duration in seconds",
			Buckets: prometheus.DefBuckets,
		}),
		ProfileDetectionTotal: mustCounterVec(factory, prometheus.CounterOpts{
			Name: "collector_profile_detection_total",
			Help: "Total profile detections by bounded profile and match kind",
		}, []string{"profile", "match_kind"}),
		ProfileFallbackTotal: mustCounter(factory, prometheus.CounterOpts{
			Name: "collector_profile_fallback_total",
			Help: "Total core-only profile fallbacks",
		}),
		ProfileCollectionFailure: mustCounterVec(factory, prometheus.CounterOpts{
			Name: "collector_profile_collection_failure_total",
			Help: "Total vendor profile collection failures",
		}, []string{"profile", "error_class"}),
		ProfileDuration: mustHistogram(factory, prometheus.HistogramOpts{
			Name:    "collector_profile_duration_seconds",
			Help:    "Vendor profile collection duration in seconds",
			Buckets: prometheus.DefBuckets,
		}),
		InterfaceSelectionTotal: mustCounterVec(factory, prometheus.CounterOpts{
			Name: "collector_interface_selection_total",
			Help: "Total interfaces annotated by bounded filter outcome",
		}, []string{"outcome"}),
		DiscoveryAttemptsTotal: mustCounter(factory, prometheus.CounterOpts{
			Name: "collector_discovery_attempts_total",
			Help: "Total discovery probe attempts",
		}),
		DiscoveryCandidatesTotal: mustCounter(factory, prometheus.CounterOpts{
			Name: "collector_discovery_candidates_total",
			Help: "Total successful discovery candidates",
		}),
		DiscoveryErrorsTotal: mustCounter(factory, prometheus.CounterOpts{
			Name: "collector_discovery_errors_total",
			Help: "Total discovery probe errors",
		}),
		DiscoveryRateLimitWaits: mustCounter(factory, prometheus.CounterOpts{
			Name: "collector_discovery_rate_limit_waits_total",
			Help: "Total discovery probes delayed by rate limiting",
		}),
		ConfigReloadSuccessTotal: mustCounter(factory, prometheus.CounterOpts{
			Name: "collector_config_reload_success_total",
			Help: "Total number of successful configuration reloads",
		}),
		ConfigReloadFailureTotal: mustCounter(factory, prometheus.CounterOpts{
			Name: "collector_config_reload_failure_total",
			Help: "Total number of failed configuration reloads",
		}),
		HealthTransitionsTotal: mustCounterVec(factory, prometheus.CounterOpts{
			Name: "collector_health_transitions_total",
			Help: "Total committed health state transitions",
		}, []string{"state", "reason", "transition"}),
		HealthDevices: mustGaugeVec(factory, prometheus.GaugeOpts{
			Name: "collector_health_devices",
			Help: "Current devices by committed terminal health state",
		}, []string{"state"}),
		DependencyImpactedDevices: mustGauge(factory, prometheus.GaugeOpts{
			Name: "collector_dependency_impacted_devices",
			Help: "Current devices in unknown/upstream_unreachable state",
		}),
		HealthPendingFailures: mustGauge(factory, prometheus.GaugeOpts{
			Name: "collector_health_pending_failures",
			Help: "Devices with pending failures below the critical threshold",
		}),
		Ready: mustGauge(factory, prometheus.GaugeOpts{
			Name: "collector_ready",
			Help: "Whether the collector is ready (1) or not (0)",
		}),
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
		HeartbeatPublishTotal: mustCounter(factory, prometheus.CounterOpts{
			Name: "collector_heartbeat_publish_total",
			Help: "Total successful collector heartbeat publishes",
		}),
		HeartbeatPublishFailure: mustCounter(factory, prometheus.CounterOpts{
			Name: "collector_heartbeat_publish_failure_total",
			Help: "Total failed collector heartbeat publishes",
		}),
		HeartbeatDuration: mustHistogram(factory, prometheus.HistogramOpts{
			Name:    "collector_heartbeat_duration_seconds",
			Help:    "Collector heartbeat publish duration in seconds",
			Buckets: prometheus.DefBuckets,
		}),
	}
	c.BufferDepth.Set(0)
	c.MQTTConnected.Set(0)
	c.Ready.Set(0)
	c.DependencyImpactedDevices.Set(0)
	c.HealthPendingFailures.Set(0)
	for _, state := range []string{"healthy", "warning", "critical", "unknown"} {
		c.HealthDevices.WithLabelValues(state).Set(0)
	}
	return c
}

// SetBufferDepth updates the Prometheus gauge from the in-memory depth counter.
func (c *Collector) SetBufferDepth(depth int64) {
	c.BufferDepth.Set(float64(depth))
}

// SetReady updates the readiness gauge (1=ready, 0=not ready).
func (c *Collector) SetReady(ready bool) {
	if ready {
		c.Ready.Set(1)
		return
	}
	c.Ready.Set(0)
}

// ObserveHealthTransition increments the health transition counter.
func (c *Collector) ObserveHealthTransition(state, reason, transition string) {
	c.HealthTransitionsTotal.WithLabelValues(state, reason, transition).Inc()
}

// SetHealthSnapshot updates gauges from committed tracker state only.
func (c *Collector) SetHealthSnapshot(devicesByState map[string]float64, dependencyImpacted, pendingFailures float64) {
	for state, value := range devicesByState {
		c.HealthDevices.WithLabelValues(state).Set(value)
	}
	c.DependencyImpactedDevices.Set(dependencyImpacted)
	c.HealthPendingFailures.Set(pendingFailures)
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

func mustGaugeVec(reg prometheus.Registerer, opts prometheus.GaugeOpts, labels []string) *prometheus.GaugeVec {
	g := prometheus.NewGaugeVec(opts, labels)
	reg.MustRegister(g)
	return g
}

func mustHistogram(reg prometheus.Registerer, opts prometheus.HistogramOpts) prometheus.Histogram {
	h := prometheus.NewHistogram(opts)
	reg.MustRegister(h)
	return h
}

// ErrorClassTimeout is used when an SNMP operation times out.
const ErrorClassTimeout = "timeout"

// ErrorClassSNMP is used for SNMP protocol / transport errors.
const ErrorClassSNMP = "snmp_error"

// ErrorClassParse is used when SNMP responses cannot be parsed.
const ErrorClassParse = "parse_error"
