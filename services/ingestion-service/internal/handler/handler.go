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
	PersistDeviceTelemetry(ctx context.Context, sample transform.DeviceTelemetrySample) (store.Result, error)
	PersistInterfaceTelemetry(ctx context.Context, sample transform.InterfaceTelemetrySample) (store.Result, error)
	PersistHealth(ctx context.Context, sample transform.HealthSample) (store.Result, error)
	PersistHeartbeat(ctx context.Context, sample transform.HeartbeatSample) (store.Result, error)
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
	case validate.KindDeviceV2:
		return h.handleDeviceV2(ctx, msg.DeviceV2)
	case validate.KindInterfaceV2:
		return h.handleInterfaceV2(ctx, msg.InterfaceV2)
	case validate.KindHealthV2:
		return h.handleHealthV2(ctx, msg.HealthV2)
	case validate.KindHeartbeatV2:
		return h.handleHeartbeatV2(ctx, msg.HeartbeatV2)
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

func (h *Handler) handleDeviceV2(ctx context.Context, msg *validate.DeviceTelemetryV2) bool {
	sample := transform.DeviceTelemetryFromValidated(*msg)
	result, err := h.store.PersistDeviceTelemetry(ctx, sample)
	if err != nil {
		if errors.Is(err, store.ErrUnknownMetricType) {
			h.metrics.MessagesRejected.Inc()
			h.log.Warn("rejected",
				"site_id", msg.Envelope.SiteID,
				"device_id", msg.Envelope.DeviceID,
				"event_id", msg.Envelope.EventID.String(),
				"result", "rejected",
				"err", err,
			)
			return true
		}
		h.metrics.DBWriteFailure.Inc()
		h.log.Error("database_error",
			"site_id", msg.Envelope.SiteID,
			"device_id", msg.Envelope.DeviceID,
			"event_id", msg.Envelope.EventID.String(),
			"result", "database_error",
			"err", err,
		)
		return false
	}
	return h.recordResult(msg.Envelope.SiteID, msg.Envelope.DeviceID, "device_telemetry", result)
}

func (h *Handler) handleInterfaceV2(ctx context.Context, msg *validate.InterfaceTelemetryV2) bool {
	sample := transform.InterfaceTelemetryFromValidated(*msg)
	result, err := h.store.PersistInterfaceTelemetry(ctx, sample)
	if err != nil {
		h.metrics.DBWriteFailure.Inc()
		h.log.Error("database_error",
			"site_id", msg.Envelope.SiteID,
			"device_id", msg.Envelope.DeviceID,
			"event_id", msg.Envelope.EventID.String(),
			"result", "database_error",
			"err", err,
		)
		return false
	}
	return h.recordResult(msg.Envelope.SiteID, msg.Envelope.DeviceID, "interface_telemetry", result)
}

func (h *Handler) handleHealthV2(ctx context.Context, msg *validate.HealthStateV2) bool {
	sample := transform.HealthFromValidated(*msg)
	result, err := h.store.PersistHealth(ctx, sample)
	if err != nil {
		h.metrics.DBWriteFailure.Inc()
		h.log.Error("database_error",
			"site_id", msg.Envelope.SiteID,
			"device_id", msg.Envelope.DeviceID,
			"event_id", msg.Envelope.EventID.String(),
			"result", "database_error",
			"err", err,
		)
		return false
	}
	return h.recordResult(msg.Envelope.SiteID, msg.Envelope.DeviceID, "health_state", result)
}

func (h *Handler) handleHeartbeatV2(ctx context.Context, msg *validate.HeartbeatV2) bool {
	sample := transform.HeartbeatFromValidated(*msg)
	result, err := h.store.PersistHeartbeat(ctx, sample)
	if err != nil {
		h.metrics.DBWriteFailure.Inc()
		h.log.Error("database_error",
			"site_id", msg.Envelope.SiteID,
			"collector_id", msg.Envelope.CollectorID,
			"event_id", msg.Envelope.EventID.String(),
			"result", "database_error",
			"err", err,
		)
		return false
	}
	return h.recordResult(msg.Envelope.SiteID, msg.Envelope.CollectorID, "collector_heartbeat", result)
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
