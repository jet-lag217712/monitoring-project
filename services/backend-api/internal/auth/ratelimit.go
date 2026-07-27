package auth

import (
	"sync"
	"time"
)

// LoginRateLimiter tracks failed login attempts per key within a sliding window.
type LoginRateLimiter struct {
	mu         sync.Mutex
	limit      int
	window     time.Duration
	maxEntries int
	entries    map[string]*loginRateEntry
	now        func() time.Time
}

type loginRateEntry struct {
	failures  int
	windowEnd time.Time
}

// NewLoginRateLimiter creates a bounded in-memory login rate limiter.
func NewLoginRateLimiter(limit int, window time.Duration, maxEntries int) *LoginRateLimiter {
	if limit <= 0 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	if maxEntries <= 0 {
		maxEntries = 1024
	}
	return &LoginRateLimiter{
		limit:      limit,
		window:     window,
		maxEntries: maxEntries,
		entries:    make(map[string]*loginRateEntry),
		now:        time.Now,
	}
}

// Allow reports whether another login attempt is permitted for key.
func (l *LoginRateLimiter) Allow(key string) bool {
	if key == "" {
		return true
	}
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneExpiredLocked(now)
	entry, ok := l.entries[key]
	if !ok {
		return true
	}
	if now.After(entry.windowEnd) {
		delete(l.entries, key)
		return true
	}
	return entry.failures < l.limit
}

// RecordFailure increments the failed attempt counter for key.
func (l *LoginRateLimiter) RecordFailure(key string) {
	if key == "" {
		return
	}
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneExpiredLocked(now)
	entry, ok := l.entries[key]
	if !ok || now.After(entry.windowEnd) {
		l.ensureCapacityLocked()
		entry = &loginRateEntry{windowEnd: now.Add(l.window)}
		l.entries[key] = entry
	}
	entry.failures++
}

// Reset clears rate-limit state for key after a successful login.
func (l *LoginRateLimiter) Reset(key string) {
	if key == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

func (l *LoginRateLimiter) pruneExpiredLocked(now time.Time) {
	for key, entry := range l.entries {
		if now.After(entry.windowEnd) {
			delete(l.entries, key)
		}
	}
}

func (l *LoginRateLimiter) ensureCapacityLocked() {
	if len(l.entries) < l.maxEntries {
		return
	}
	var oldestKey string
	var oldestEnd time.Time
	first := true
	for key, entry := range l.entries {
		if first || entry.windowEnd.Before(oldestEnd) {
			oldestKey = key
			oldestEnd = entry.windowEnd
			first = false
		}
	}
	if oldestKey != "" {
		delete(l.entries, oldestKey)
	}
}
