// Package profile defines vendor enrichment and deterministic detection.
package profile

import (
	"context"
	"strings"

	"github.com/gosnmp/gosnmp"

	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/readings"
)

// Client is the SNMP surface available to vendor profiles.
type Client interface {
	Get(ctx context.Context, oids []string) (*gosnmp.SnmpPacket, error)
	Walk(ctx context.Context, rootOID string, walkFn gosnmp.WalkFunc) error
}

// Profile enriches a core result without mutating its identity or interfaces.
type Profile interface {
	Name() string
	Capabilities() readings.Capability
	ExactObjectIDs() []string
	ObjectIDPrefixes() []string
	GenericVendorPrefix() string
	Collect(ctx context.Context, client Client) (readings.VendorReadings, error)
}

// MatchKind identifies the deterministic detection tier.
type MatchKind string

const (
	MatchExact   MatchKind = "exact"
	MatchPrefix  MatchKind = "prefix"
	MatchGeneric MatchKind = "generic_vendor"
	MatchCore    MatchKind = "core"
)

// Registry selects profiles using only sysObjectID.
type Registry struct {
	profiles []Profile
}

// NewRegistry creates an immutable profile registry.
func NewRegistry(profiles ...Profile) *Registry {
	return &Registry{profiles: append([]Profile(nil), profiles...)}
}

// Match applies exact, longest model prefix, generic vendor, then core precedence.
func (r *Registry) Match(sysObjectID string) (Profile, MatchKind) {
	oid := normalizeOID(sysObjectID)
	for _, candidate := range r.profiles {
		for _, exact := range candidate.ExactObjectIDs() {
			if oid == normalizeOID(exact) {
				return candidate, MatchExact
			}
		}
	}

	var longest Profile
	longestLength := -1
	for _, candidate := range r.profiles {
		for _, prefix := range candidate.ObjectIDPrefixes() {
			normalized := normalizeOID(prefix)
			if oidHasPrefix(oid, normalized) && len(normalized) > longestLength {
				longest = candidate
				longestLength = len(normalized)
			}
		}
	}
	if longest != nil {
		return longest, MatchPrefix
	}

	longest = nil
	longestLength = -1
	for _, candidate := range r.profiles {
		prefix := normalizeOID(candidate.GenericVendorPrefix())
		if prefix != "" && oidHasPrefix(oid, prefix) && len(prefix) > longestLength {
			longest = candidate
			longestLength = len(prefix)
		}
	}
	if longest != nil {
		return longest, MatchGeneric
	}
	return nil, MatchCore
}

func oidHasPrefix(oid, prefix string) bool {
	return oid == prefix || strings.HasPrefix(oid, prefix+".")
}

func normalizeOID(oid string) string {
	return strings.TrimLeft(strings.TrimSpace(oid), ".")
}
