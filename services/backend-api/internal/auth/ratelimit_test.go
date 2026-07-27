package auth_test

import (
	"testing"
	"time"

	"github.com/equate/ogsd/services/backend-api/internal/auth"
)

func TestLoginRateLimiterEvictsOldestEntry(t *testing.T) {
	limiter := auth.NewLoginRateLimiter(1, time.Minute, 2)
	limiter.RecordFailure("a")
	limiter.RecordFailure("b")
	if !limiter.Allow("c") {
		t.Fatal("expected eviction to keep limiter usable")
	}
}

func TestLoginRateLimiterEmptyKeyAllowed(t *testing.T) {
	limiter := auth.NewLoginRateLimiter(1, time.Minute, 2)
	if !limiter.Allow("") {
		t.Fatal("expected empty key to be allowed")
	}
}
