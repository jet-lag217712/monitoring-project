package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/equate/ogsd/services/snmp-collector/internal/events"
)

// StdoutPublisher writes one JSON object per line to an io.Writer (default stdout).
type StdoutPublisher struct {
	w  io.Writer
	mu sync.Mutex
}

// NewStdoutPublisher returns a publisher that writes JSON lines to stdout.
func NewStdoutPublisher() *StdoutPublisher {
	return &StdoutPublisher{w: os.Stdout}
}

// NewStdoutPublisherWriter returns a publisher writing to w (useful in tests).
func NewStdoutPublisherWriter(w io.Writer) *StdoutPublisher {
	return &StdoutPublisher{w: w}
}

// Publish encodes each event as a single JSON line.
func (p *StdoutPublisher) Publish(ctx context.Context, evs ...events.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	enc := json.NewEncoder(p.w)
	for _, ev := range evs {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := enc.Encode(ev); err != nil {
			return fmt.Errorf("encode event: %w", err)
		}
	}
	return nil
}
