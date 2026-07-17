package profile

import (
	"context"
	"testing"

	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/readings"
)

type fakeProfile struct {
	name     string
	exact    []string
	prefixes []string
	generic  string
}

func (p fakeProfile) Name() string                      { return p.name }
func (p fakeProfile) Capabilities() readings.Capability { return 0 }
func (p fakeProfile) ExactObjectIDs() []string          { return p.exact }
func (p fakeProfile) ObjectIDPrefixes() []string        { return p.prefixes }
func (p fakeProfile) GenericVendorPrefix() string       { return p.generic }
func (p fakeProfile) Collect(context.Context, Client) (readings.VendorReadings, error) {
	return readings.VendorReadings{}, nil
}

func TestRegistryMatchPrecedence(t *testing.T) {
	t.Parallel()

	exact := fakeProfile{name: "exact", exact: []string{"1.3.6.1.4.1.9.1.100"}}
	short := fakeProfile{name: "short", prefixes: []string{"1.3.6.1.4.1.9.1"}}
	long := fakeProfile{name: "long", prefixes: []string{"1.3.6.1.4.1.9.1.200"}}
	generic := fakeProfile{name: "generic", generic: "1.3.6.1.4.1.9"}
	registry := NewRegistry(generic, short, long, exact)

	tests := []struct {
		name string
		oid  string
		want string
		kind MatchKind
	}{
		{name: "exact", oid: ".1.3.6.1.4.1.9.1.100", want: "exact", kind: MatchExact},
		{name: "longest prefix", oid: "1.3.6.1.4.1.9.1.200.7", want: "long", kind: MatchPrefix},
		{name: "generic vendor", oid: "1.3.6.1.4.1.9.99.1", want: "generic", kind: MatchGeneric},
		{name: "core", oid: "1.3.6.1.4.1.55555.1", kind: MatchCore},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, kind := registry.Match(tt.oid)
			if kind != tt.kind {
				t.Fatalf("kind=%q, want %q", kind, tt.kind)
			}
			if tt.want == "" {
				if got != nil {
					t.Fatalf("profile=%q, want nil", got.Name())
				}
				return
			}
			if got == nil || got.Name() != tt.want {
				t.Fatalf("profile=%v, want %q", got, tt.want)
			}
		})
	}
}

func TestRegistryNeverUsesSysDescr(t *testing.T) {
	t.Parallel()

	registry := NewRegistry(fakeProfile{name: "cisco", generic: "1.3.6.1.4.1.9"})
	got, kind := registry.Match("1.3.6.1.4.1.55555.1")
	if got != nil || kind != MatchCore {
		t.Fatalf("unexpected match from non-Cisco object ID: %v %q", got, kind)
	}
}
