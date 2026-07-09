package publisher

import (
	"testing"
	"time"
)

func TestJitterBackoffBounds(t *testing.T) {
	t.Parallel()
	fn := jitterBackoff(time.Second, 8*time.Second)

	for attempt := 0; attempt < 10; attempt++ {
		d := fn(attempt)
		if d < time.Second {
			t.Fatalf("attempt %d: got %v < initial", attempt, d)
		}
		// base is min(initial*2^attempt, max); jitter adds up to base/2.
		maxExpected := 8*time.Second + 4*time.Second
		if d > maxExpected {
			t.Fatalf("attempt %d: got %v > %v", attempt, d, maxExpected)
		}
	}
}

func TestJitterBackoffIncreases(t *testing.T) {
	t.Parallel()
	fn := jitterBackoff(time.Second, 60*time.Second)
	// Without jitter the sequence is 1s, 2s, 4s...; with jitter, attempt 3 should
	// still be larger than attempt 0 on average. Check a hard lower bound.
	d0 := fn(0)
	d3 := fn(3)
	if d3 < d0 {
		t.Fatalf("expected later attempt >= earlier lower bound: d0=%v d3=%v", d0, d3)
	}
}
