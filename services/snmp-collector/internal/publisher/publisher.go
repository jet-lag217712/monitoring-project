package publisher

import (
	"context"

	"github.com/equate/ogsd/services/snmp-collector/internal/events"
)

// Publisher delivers normalized telemetry events.
// Phase 1 uses StdoutPublisher; Phase 2 swaps in a buffered MQTT implementation.
type Publisher interface {
	Publish(ctx context.Context, evs ...events.Event) error
}
