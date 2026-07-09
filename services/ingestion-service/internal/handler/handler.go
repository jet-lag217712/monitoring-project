package handler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/equate/ogsd/services/ingestion-service/internal/metrics"
	"github.com/equate/ogsd/services/ingestion-service/internal/store"
	"github.com/equate/ogsd/services/ingestion-service/internal/transform"
	"github.com/equate/ogsd/services/ingestion-service/internal/validate"
)

// Persister is the store boundary used by the handler (tests inject fakes).
type Persister interface {
	PersistDeviceSample(ctx context.Context, sample transform.DeviceSample) (store.Result, error)
	PersistInterfaceSample(ctx context.Context, sample transform.InterfaceSample) (store.Result, error)
}

// Handler orchestrates validate → transform → persist → ACK decision.
type Handler struct {
	store   Persister
	metrics *metrics.Ingestion
	log     *slog.Logger
}

// New creates a message handler.
func New(s Persister, m *metrics.Ingestion, log *slog.Logger) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{store: s, metrics: m, log: log}
}

// Handle processes one MQTT message. Returns ack=true when the broker may
// discard the message (accepted, duplicate, or rejected). Returns ack=false
// on database failure so the broker redelivers.
func (h *Handler) Handle(ctx context.Context, topic string, payload []byte) (ack bool) {
	start := time.Now()
	defer func() {
		h.metrics.ProcessingDuration.Observe(time.Since(start).Seconds())
	}()
	h.metrics.MessagesReceived.Inc()

	msg, err := validate.Validate(topic, payload)
	if err != nil {
		h.metrics.MessagesRejected.Inc()
		h.log.Warn("rejected", "topic", topic, "result", "rejected", "err", err)
		return true
	}

	switch msg.Kind {
	case validate.KindDevice:
		return h.handleDevice(ctx, msg.Device)
	case validate.KindInterface:
		return h.handleInterface(ctx, msg.Interface)
	default:
		h.metrics.MessagesRejected.Inc()
		h.log.Warn("rejected", "topic", topic, "result", "rejected", "err", "unknown kind")
		return true
	}
}

func (h *Handler) handleDevice(ctx context.Context, msg *validate.DeviceMessage) bool {
	sample := transform.DeviceSampleFromValidated(*msg)
	result, err := h.store.PersistDeviceSample(ctx, sample)
	if err != nil {
		if errors.Is(err, store.ErrUnknownMetricType) {
			h.metrics.MessagesRejected.Inc()
			h.log.Warn("rejected",
				"site_id", msg.SiteID,
				"device_id", msg.DeviceID,
				"metric", msg.Metric,
				"result", "rejected",
				"err", err,
			)
			return true
		}
		h.metrics.DBWriteFailure.Inc()
		h.log.Error("database_error",
			"site_id", msg.SiteID,
			"device_id", msg.DeviceID,
			"metric", msg.Metric,
			"result", "database_error",
			"err", err,
		)
		return false
	}
	return h.recordResult(msg.SiteID, msg.DeviceID, msg.Metric, result)
}

func (h *Handler) handleInterface(ctx context.Context, msg *validate.InterfaceMessage) bool {
	sample := transform.InterfaceSampleFromValidated(*msg)
	result, err := h.store.PersistInterfaceSample(ctx, sample)
	if err != nil {
		h.metrics.DBWriteFailure.Inc()
		h.log.Error("database_error",
			"site_id", msg.SiteID,
			"device_id", msg.DeviceID,
			"if_index", msg.IfIndex,
			"result", "database_error",
			"err", err,
		)
		return false
	}
	return h.recordResult(msg.SiteID, msg.DeviceID, "interface", result)
}

func (h *Handler) recordResult(siteID, deviceID, metric string, result store.Result) bool {
	switch result {
	case store.ResultDuplicate:
		h.metrics.MessagesDeduplicated.Inc()
		h.log.Info("deduplicated",
			"site_id", siteID,
			"device_id", deviceID,
			"metric", metric,
			"result", "deduplicated",
		)
	default:
		h.metrics.MessagesAccepted.Inc()
		h.log.Info("accepted",
			"site_id", siteID,
			"device_id", deviceID,
			"metric", metric,
			"result", "accepted",
		)
	}
	return true
}
