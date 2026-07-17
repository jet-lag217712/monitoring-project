package discovery

import (
	"context"
	"sync"
	"time"
)

type sleepFunc func(context.Context, time.Duration) error

type tokenBucket struct {
	mu sync.Mutex

	rate   float64
	burst  float64
	tokens float64
	last   time.Time

	now   func() time.Time
	sleep sleepFunc
}

func newTokenBucket(rate float64, burst int) *tokenBucket {
	return newTokenBucketWithClock(rate, burst, time.Now, sleepContext)
}

func newTokenBucketWithClock(rate float64, burst int, now func() time.Time, sleep sleepFunc) *tokenBucket {
	current := now()
	return &tokenBucket{
		rate:   rate,
		burst:  float64(burst),
		tokens: float64(burst),
		last:   current,
		now:    now,
		sleep:  sleep,
	}
}

// Wait reserves one token while holding the mutex. Concurrent workers may wait
// independently, but every reservation is ordered through the same bucket.
// delayed is true when the caller had to sleep for a rate-limit deficit.
func (bucket *tokenBucket) Wait(ctx context.Context) (delayed bool, err error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	bucket.mu.Lock()
	now := bucket.now()
	if elapsed := now.Sub(bucket.last).Seconds(); elapsed > 0 {
		bucket.tokens += elapsed * bucket.rate
		if bucket.tokens > bucket.burst {
			bucket.tokens = bucket.burst
		}
		bucket.last = now
	}
	bucket.tokens--
	delay := time.Duration(0)
	if bucket.tokens < 0 {
		delay = time.Duration((-bucket.tokens / bucket.rate) * float64(time.Second))
	}
	bucket.mu.Unlock()

	if delay <= 0 {
		return false, nil
	}
	return true, bucket.sleep(ctx, delay)
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
