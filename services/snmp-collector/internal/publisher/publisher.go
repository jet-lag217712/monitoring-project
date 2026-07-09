package publisher

import (
	"context"

	"github.com/equate/ogsd/services/snmp-collector/internal/events"
)

// Publisher delivers normalized telemetry events.
// Implementations: StdoutPublisher (dev) and BufferedPublisher (MQTT/TLS).
type Publisher interface {
	Publish(ctx context.Context, evs ...events.Event) error
}
