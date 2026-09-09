// Package discovery performs operator-invoked, allowlisted SNMP discovery.
package discovery

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/netip"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
)

const (
	// SysObjectIDOID is the SNMPv2-MIB sysObjectID scalar.
	SysObjectIDOID = "1.3.6.1.2.1.1.2.0"
	// SysNameOID is the SNMPv2-MIB sysName scalar.
	SysNameOID = "1.3.6.1.2.1.1.5.0"
	// SysDescrOID is the SNMPv2-MIB sysDescr scalar.
	SysDescrOID = "1.3.6.1.2.1.1.1.0"
)

var identityOIDs = [3]string{SysObjectIDOID, SysNameOID, SysDescrOID}

// ProbeRequest is the complete, read-only SNMP request allowed during discovery.
type ProbeRequest struct {
	IP        netip.Addr
	Community string
	Timeout   time.Duration
	Retries   int
	OIDs      [3]string
}

// Identity is the limited SNMP identity returned by a discovery probe.
type Identity struct {
	SysObjectID string
	SysName     string
	SysDescr    string
}

// Prober performs one SNMPv2c identity probe. Implementations must not use ICMP
// or issue SNMP writes.
type Prober interface {
	Probe(ctx context.Context, request ProbeRequest) (Identity, error)
}

// ProfileDetector maps an identity to a configured profile name.
type ProfileDetector func(identity Identity) string

// ProbeResult records whether a target returned a valid identity.
type ProbeResult string

const (
	// ProbeSucceeded means the target returned sysObjectID.
	ProbeSucceeded ProbeResult = "success"
	// ProbeFailed means the target did not return a valid identity.
	ProbeFailed ProbeResult = "error"
)

// Candidate is a reviewable discovery result. It never contains the community.
type Candidate struct {
	IP              string      `json:"ip" yaml:"ip"`
	Fingerprint     string      `json:"fingerprint,omitempty" yaml:"fingerprint,omitempty"`
	DetectedProfile string      `json:"detected_profile,omitempty" yaml:"detected_profile,omitempty"`
	Hostname        string      `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	Description     string      `json:"description,omitempty" yaml:"description,omitempty"`
	Result          ProbeResult `json:"result" yaml:"result"`
	Error           string      `json:"error,omitempty" yaml:"error,omitempty"`
	Timestamp       time.Time   `json:"timestamp" yaml:"timestamp"`
}

// Option customizes a Scanner.
type Option func(*Scanner)

// WithProfileDetector configures profile detection for successful probes.
func WithProfileDetector(detector ProfileDetector) Option {
	return func(scanner *Scanner) {
		scanner.detector = detector
	}
}

// ProbeProgressFunc reports probe completion counts during Scan.
type ProbeProgressFunc func(probed, total int)

// WithProbeProgress configures a callback invoked after each target is probed.
func WithProbeProgress(fn ProbeProgressFunc) Option {
	return func(scanner *Scanner) {
		scanner.onProbeComplete = fn
	}
}

type probeWaiter interface {
	Wait(context.Context) (delayed bool, err error)
}

// Scanner expands and probes only the CIDRs in its immutable discovery policy.
type Scanner struct {
	policy          config.DiscoveryConfig
	community       string
	prober          Prober
	detector        ProfileDetector
	limiter         probeWaiter
	now             func() time.Time
	onRateLimitWait func()
	onProbeComplete ProbeProgressFunc
}

// WithRateLimitWaitObserver records probes delayed by the shared token bucket.
func WithRateLimitWaitObserver(observer func()) Option {
	return func(scanner *Scanner) {
		scanner.onRateLimitWait = observer
	}
}

// New creates a scanner with a shared token bucket independent of worker count.
func New(policy config.DiscoveryConfig, community string, prober Prober, options ...Option) (*Scanner, error) {
	if err := validatePolicy(policy); err != nil {
		return nil, err
	}
	if strings.TrimSpace(community) == "" {
		return nil, errors.New("SNMP community is required")
	}
	if prober == nil {
		return nil, errors.New("prober is required")
	}

	policy.AllowedCIDRs = append([]string(nil), policy.AllowedCIDRs...)
	scanner := &Scanner{
		policy:    policy,
		community: community,
		prober:    prober,
		limiter:   newTokenBucket(policy.MaxProbesPerSecond, policy.ProbeBurst),
		now:       time.Now,
	}
	for _, option := range options {
		if option != nil {
			option(scanner)
		}
	}
	return scanner, nil
}

// Scan expands every allowed CIDR before starting any probe, then uses a
// bounded worker pool whose concurrency cannot bypass the shared rate limiter.
func (s *Scanner) Scan(ctx context.Context) ([]Candidate, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	targets, err := expandAllowedTargets(ctx, s.policy.AllowedCIDRs, s.policy.MaxTargets)
	if err != nil {
		return nil, err
	}

	workers := s.policy.MaxWorkers
	if workers > len(targets) {
		workers = len(targets)
	}
	total := len(targets)
	if s.onProbeComplete != nil {
		s.onProbeComplete(0, total)
	}
	jobs := make(chan netip.Addr)
	results := make(chan Candidate, len(targets))
	var probed atomic.Int64

	var workersWG sync.WaitGroup
	workersWG.Add(workers)
	for range workers {
		go func() {
			defer workersWG.Done()
			for target := range jobs {
				delayed, err := s.limiter.Wait(ctx)
				if err != nil {
					continue
				}
				if delayed && s.onRateLimitWait != nil {
					s.onRateLimitWait()
				}
				results <- s.probe(ctx, target)
				n := probed.Add(1)
				if s.onProbeComplete != nil {
					s.onProbeComplete(int(n), total)
				}
			}
		}()
	}

sendTargets:
	for _, target := range targets {
		select {
		case jobs <- target:
		case <-ctx.Done():
			break sendTargets
		}
	}
	close(jobs)
	workersWG.Wait()
	close(results)

	candidates := make([]Candidate, 0, len(results))
	for candidate := range results {
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool {
		left, _ := netip.ParseAddr(candidates[i].IP)
		right, _ := netip.ParseAddr(candidates[j].IP)
		return left.Compare(right) < 0
	})
	if err := ctx.Err(); err != nil {
		return candidates, err
	}
	return candidates, nil
}

// TargetCount returns how many addresses would be probed for the given policy.
func TargetCount(ctx context.Context, policy config.DiscoveryConfig) (int, error) {
	targets, err := expandAllowedTargets(ctx, policy.AllowedCIDRs, policy.MaxTargets)
	if err != nil {
		return 0, err
	}
	return len(targets), nil
}

func (s *Scanner) probe(ctx context.Context, target netip.Addr) Candidate {
	timeout := s.policy.Timeout * time.Duration(s.policy.Retries+1)
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	identity, err := s.prober.Probe(probeCtx, ProbeRequest{
		IP:        target,
		Community: s.community,
		Timeout:   s.policy.Timeout,
		Retries:   s.policy.Retries,
		OIDs:      identityOIDs,
	})
	candidate := Candidate{
		IP:        target.String(),
		Result:    ProbeFailed,
		Timestamp: s.now().UTC(),
	}
	if err != nil {
		candidate.Error = redactError(err, s.community)
		return candidate
	}

	identity.SysObjectID = normalizeOID(identity.SysObjectID)
	if identity.SysObjectID == "" {
		candidate.Error = "probe returned an empty sysObjectID"
		return candidate
	}
	candidate.Fingerprint = identity.SysObjectID
	candidate.Hostname = strings.TrimSpace(identity.SysName)
	candidate.Description = strings.TrimSpace(identity.SysDescr)
	candidate.DetectedProfile = "core"
	if s.detector != nil {
		if profile := strings.TrimSpace(s.detector(identity)); profile != "" {
			candidate.DetectedProfile = profile
		}
	}
	candidate.Result = ProbeSucceeded
	return candidate
}

func validatePolicy(policy config.DiscoveryConfig) error {
	if len(policy.AllowedCIDRs) == 0 {
		return errors.New("at least one allowed CIDR is required")
	}
	if policy.MaxTargets <= 0 {
		return errors.New("max_targets must be positive")
	}
	if policy.Timeout <= 0 {
		return errors.New("timeout must be positive")
	}
	if policy.Retries < 0 {
		return errors.New("retries must not be negative")
	}
	if policy.MaxWorkers <= 0 {
		return errors.New("max_workers must be positive")
	}
	if policy.MaxProbesPerSecond <= 0 || math.IsNaN(policy.MaxProbesPerSecond) || math.IsInf(policy.MaxProbesPerSecond, 0) {
		return errors.New("max_probes_per_second must be finite and positive")
	}
	if policy.ProbeBurst <= 0 {
		return errors.New("probe_burst must be positive")
	}
	if policy.Timeout > time.Duration(math.MaxInt64)/time.Duration(policy.Retries+1) {
		return errors.New("timeout and retries exceed the supported probe duration")
	}
	for index, cidr := range policy.AllowedCIDRs {
		if _, err := netip.ParsePrefix(strings.TrimSpace(cidr)); err != nil {
			return fmt.Errorf("allowed_cidrs[%d]: %w", index, err)
		}
	}
	return nil
}

func expandAllowedTargets(ctx context.Context, cidrs []string, maxTargets int) ([]netip.Addr, error) {
	targets := make([]netip.Addr, 0, min(maxTargets, 256))
	seen := make(map[netip.Addr]struct{}, min(maxTargets, 256))
	for _, cidr := range cidrs {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(cidr))
		if err != nil {
			return nil, fmt.Errorf("parse allowed CIDR %q: %w", cidr, err)
		}
		prefix = prefix.Masked()
		for address := prefix.Addr(); address.IsValid() && prefix.Contains(address); address = address.Next() {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			address = address.Unmap()
			if _, exists := seen[address]; exists {
				continue
			}
			if len(targets) == maxTargets {
				return nil, fmt.Errorf("allowed CIDRs expand beyond max_targets (%d)", maxTargets)
			}
			seen[address] = struct{}{}
			targets = append(targets, address)
		}
	}
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Compare(targets[j]) < 0
	})
	return targets, nil
}

func normalizeOID(oid string) string {
	return strings.TrimLeft(strings.TrimSpace(oid), ".")
}

func redactError(err error, secret string) string {
	message := err.Error()
	if secret != "" {
		message = strings.ReplaceAll(message, secret, "[redacted]")
	}
	return message
}
