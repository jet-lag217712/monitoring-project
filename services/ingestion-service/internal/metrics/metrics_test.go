package metrics_test

import (
	"testing"

	"github.com/equate/ogsd/services/ingestion-service/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func TestMetrics_Register(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewWithRegisterer(reg)
	if m.MessagesReceived == nil || m.MessagesAccepted == nil || m.MessagesRejected == nil {
		t.Fatal("counters nil")
	}
	if m.MessagesDeduplicated == nil || m.DBWriteFailure == nil {
		t.Fatal("counters nil")
	}
	if m.ProcessingDuration == nil || m.MQTTConnected == nil || m.MQTTSubscribed == nil {
		t.Fatal("instruments nil")
	}
	// Second registerer must not panic when using a fresh registry.
	_ = metrics.NewWithRegisterer(prometheus.NewRegistry())
}
