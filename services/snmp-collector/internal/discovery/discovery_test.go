package discovery

import (
	"context"
	"fmt"
	"net/netip"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
)

type proberFunc func(context.Context, ProbeRequest) (Identity, error)

func (function proberFunc) Probe(ctx context.Context, request ProbeRequest) (Identity, error) {
	return function(ctx, request)
}

func testPolicy(cidr string, maxTargets, maxWorkers int) config.DiscoveryConfig {
	return config.DiscoveryConfig{
		AllowedCIDRs:       []string{cidr},
		MaxTargets:         maxTargets,
		Timeout:            time.Second,
		Retries:            1,
		MaxWorkers:         maxWorkers,
		MaxProbesPerSecond: 10_000,
		ProbeBurst:         maxTargets,
	}
}

func TestScanExpandsOnlyConfiguredAllowlistAndUsesIdentityOIDs(t *testing.T) {
	var (
		mu       sync.Mutex
		requests []ProbeRequest
	)
	prober := proberFunc(func(_ context.Context, request ProbeRequest) (Identity, error) {
		mu.Lock()
		requests = append(requests, request)
		mu.Unlock()
		if request.OIDs != identityOIDs {
			return Identity{}, fmt.Errorf("unexpected OIDs: %v", request.OIDs)
		}
		if request.Community != "private-community" {
			return Identity{}, fmt.Errorf("unexpected community")
		}
		return Identity{
			SysObjectID: ".1.3.6.1.4.1.9.1",
			SysName:     " switch-1 ",
			SysDescr:    " Cisco IOS ",
		}, nil
	})
	scanner, err := New(
		testPolicy("192.0.2.0/30", 4, 2),
		"private-community",
		prober,
		WithProfileDetector(func(Identity) string { return "cisco" }),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	candidates, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	gotIPs := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		gotIPs = append(gotIPs, candidate.IP)
		if candidate.Result != ProbeSucceeded ||
			candidate.Fingerprint != "1.3.6.1.4.1.9.1" ||
			candidate.DetectedProfile != "cisco" ||
			candidate.Hostname != "switch-1" ||
			candidate.Description != "Cisco IOS" {
			t.Fatalf("candidate=%#v", candidate)
		}
	}
	wantIPs := []string{"192.0.2.0", "192.0.2.1", "192.0.2.2", "192.0.2.3"}
	if !slices.Equal(gotIPs, wantIPs) {
		t.Fatalf("probed IPs=%v, want %v", gotIPs, wantIPs)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(requests) != len(wantIPs) {
		t.Fatalf("requests=%d, want %d", len(requests), len(wantIPs))
	}
	for _, request := range requests {
		if !netip.MustParsePrefix("192.0.2.0/30").Contains(request.IP) {
			t.Fatalf("request escaped configured allowlist: %s", request.IP)
		}
	}
}

func TestScanEnforcesTargetCapBeforeAnyProbe(t *testing.T) {
	var calls atomic.Int32
	prober := proberFunc(func(context.Context, ProbeRequest) (Identity, error) {
		calls.Add(1)
		return Identity{SysObjectID: "1.3.6.1.4.1.9"}, nil
	})
	scanner, err := New(testPolicy("198.51.100.0/29", 4, 4), "secret", prober)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := scanner.Scan(context.Background()); err == nil {
		t.Fatal("Scan succeeded despite target cap")
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("probe calls=%d, want 0", got)
	}
}

type countingWaiter struct {
	waits atomic.Int32
}

func (waiter *countingWaiter) Wait(context.Context) (bool, error) {
	waiter.waits.Add(1)
	return false, nil
}

func TestScanRateLimitsBeforeEveryProbe(t *testing.T) {
	var calls atomic.Int32
	waiter := &countingWaiter{}
	prober := proberFunc(func(context.Context, ProbeRequest) (Identity, error) {
		call := calls.Add(1)
		if waiter.waits.Load() < call {
			return Identity{}, fmt.Errorf("probe %d ran before limiter", call)
		}
		return Identity{SysObjectID: "1.3.6.1.4.1.30065"}, nil
	})
	scanner, err := New(testPolicy("203.0.113.0/29", 8, 8), "secret", prober)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	scanner.limiter = waiter

	candidates, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(candidates) != 8 || calls.Load() != 8 || waiter.waits.Load() != 8 {
		t.Fatalf("candidates=%d calls=%d waits=%d", len(candidates), calls.Load(), waiter.waits.Load())
	}
	for _, candidate := range candidates {
		if candidate.Result != ProbeSucceeded {
			t.Fatalf("candidate=%#v", candidate)
		}
	}
}

func TestScanBoundsConcurrencyIndependentlyOfRate(t *testing.T) {
	const workers = 3
	started := make(chan struct{}, 8)
	release := make(chan struct{})
	released := false
	defer func() {
		if !released {
			close(release)
		}
	}()

	var (
		active    atomic.Int32
		maxActive atomic.Int32
	)
	prober := proberFunc(func(ctx context.Context, _ ProbeRequest) (Identity, error) {
		current := active.Add(1)
		defer active.Add(-1)
		for {
			maximum := maxActive.Load()
			if current <= maximum || maxActive.CompareAndSwap(maximum, current) {
				break
			}
		}
		started <- struct{}{}
		select {
		case <-release:
			return Identity{SysObjectID: "1.3.6.1.4.1.9"}, nil
		case <-ctx.Done():
			return Identity{}, ctx.Err()
		}
	})
	policy := testPolicy("192.0.2.0/29", 8, workers)
	policy.MaxProbesPerSecond = 1
	policy.ProbeBurst = 8
	scanner, err := New(policy, "secret", prober)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		_, scanErr := scanner.Scan(context.Background())
		done <- scanErr
	}()
	for range workers {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("workers did not reach configured concurrency")
		}
	}
	select {
	case <-started:
		t.Fatal("probe concurrency exceeded worker limit")
	case <-time.After(25 * time.Millisecond):
	}
	close(release)
	released = true
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Scan did not finish")
	}
	if got := maxActive.Load(); got != workers {
		t.Fatalf("maximum concurrency=%d, want %d", got, workers)
	}
}

func TestProbeErrorRedactsCommunity(t *testing.T) {
	prober := proberFunc(func(context.Context, ProbeRequest) (Identity, error) {
		return Identity{}, fmt.Errorf("authentication failed for top-secret")
	})
	scanner, err := New(testPolicy("192.0.2.1/32", 1, 1), "top-secret", prober)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	candidates, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(candidates) != 1 || candidates[0].Error != "authentication failed for [redacted]" {
		t.Fatalf("candidates=%#v", candidates)
	}
}
