package cisco

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/readings"
)

type fixture struct {
	Walks  map[string][]fixturePDU `json:"walks"`
	Errors map[string]string       `json:"errors"`
}

type fixturePDU struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

type fixtureClient struct {
	fixture fixture
}

func (fixtureClient) Get(context.Context, []string) (*gosnmp.SnmpPacket, error) {
	return nil, errors.New("unexpected Get")
}

func (c fixtureClient) Walk(_ context.Context, root string, walkFn gosnmp.WalkFunc) error {
	if message := c.fixture.Errors[root]; message != "" {
		return errors.New(message)
	}
	for _, item := range c.fixture.Walks[root] {
		if err := walkFn(gosnmp.SnmpPDU{
			Name:  item.Name,
			Type:  fixturePDUType(item.Type),
			Value: item.Value,
		}); err != nil {
			return err
		}
	}
	return nil
}

func TestProfileMetadata(t *testing.T) {
	t.Parallel()

	p := New()
	if p.Name() != "cisco" {
		t.Fatalf("Name()=%q", p.Name())
	}
	for _, capability := range []readings.Capability{
		readings.CapabilityCPU,
		readings.CapabilityMemory,
		readings.CapabilityTemperature,
		readings.CapabilityPower,
	} {
		if !p.Capabilities().Has(capability) {
			t.Fatalf("Capabilities() missing %v", capability)
		}
	}
	if p.GenericVendorPrefix() != "1.3.6.1.4.1.9" {
		t.Fatalf("GenericVendorPrefix()=%q", p.GenericVendorPrefix())
	}
	if len(p.ExactObjectIDs()) == 0 {
		t.Fatal("ExactObjectIDs() is empty")
	}
}

func TestCollectSuccessFixture(t *testing.T) {
	t.Parallel()

	got, err := New().Collect(context.Background(), fixtureClient{fixture: loadFixture(t, "success.json")})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if got.Profile != "cisco" || got.Capabilities != capabilities {
		t.Fatalf("metadata=%#v", got)
	}
	if got.CPU == nil || got.CPU.Value != 30 || got.CPU.SourceOID != oidCPU5Min {
		t.Fatalf("CPU=%#v", got.CPU)
	}
	if got.Memory == nil || got.Memory.Value != 40 || got.Memory.SourceOID != oidMemoryPoolUsed {
		t.Fatalf("Memory=%#v", got.Memory)
	}
	if len(got.Temperatures) != 2 ||
		got.Temperatures[0].Value == nil ||
		*got.Temperatures[0].Value != 42 ||
		got.Temperatures[1].Status != "warning" {
		t.Fatalf("Temperatures=%#v", got.Temperatures)
	}
	if len(got.Power) != 2 ||
		got.Power[0].Value != nil ||
		got.Power[0].Status != "ok" ||
		got.Power[1].Status != "not_present" {
		t.Fatalf("Power=%#v", got.Power)
	}
}

func TestCollectFailureFixtures(t *testing.T) {
	t.Parallel()

	fixtures := loadFixtureSet(t, "failures.json")
	tests := []struct {
		name  string
		check func(*testing.T, readings.VendorReadings, error)
	}{
		{
			name: "missing_subtree",
			check: func(t *testing.T, got readings.VendorReadings, err error) {
				if err != nil {
					t.Fatalf("Collect: %v", err)
				}
				if got.CPU == nil || got.Memory != nil || len(got.Temperatures) != 0 || len(got.Power) != 0 {
					t.Fatalf("readings=%#v", got)
				}
			},
		},
		{
			name: "no_such_object",
			check: func(t *testing.T, got readings.VendorReadings, err error) {
				if err != nil {
					t.Fatalf("Collect: %v", err)
				}
				if got.CPU != nil || got.Memory == nil || got.Memory.Value != 25 {
					t.Fatalf("readings=%#v", got)
				}
			},
		},
		{
			name: "timeout",
			check: func(t *testing.T, got readings.VendorReadings, err error) {
				if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
					t.Fatalf("error=%v", err)
				}
				if got.CPU == nil || got.CPU.Value != 33 ||
					len(got.Temperatures) != 1 ||
					got.Temperatures[0].Value == nil ||
					*got.Temperatures[0].Value != 41 {
					t.Fatalf("partial readings=%#v", got)
				}
			},
		},
		{
			name: "partial_tables",
			check: func(t *testing.T, got readings.VendorReadings, err error) {
				if err != nil {
					t.Fatalf("Collect: %v", err)
				}
				if got.CPU == nil || got.CPU.Value != 0 {
					t.Fatalf("valid zero CPU was not preserved: %#v", got.CPU)
				}
				if got.Memory != nil {
					t.Fatalf("incomplete memory table produced a reading: %#v", got.Memory)
				}
				if len(got.Temperatures) != 1 ||
					got.Temperatures[0].Value != nil ||
					got.Temperatures[0].Unit != "" ||
					got.Temperatures[0].Status != "warning" {
					t.Fatalf("partial temperature=%#v", got.Temperatures)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := New().Collect(context.Background(), fixtureClient{fixture: fixtures[test.name]})
			if got.Profile != "cisco" || got.Capabilities != capabilities {
				t.Fatalf("metadata=%#v", got)
			}
			test.check(t, got, err)
		})
	}
}

func loadFixture(t *testing.T, name string) fixture {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var result struct {
		Walks  map[string][]fixturePDU `json:"walks"`
		Errors map[string]string       `json:"errors"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return fixture{Walks: result.Walks, Errors: result.Errors}
}

func loadFixtureSet(t *testing.T, name string) map[string]fixture {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture set: %v", err)
	}
	var result map[string]fixture
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("decode fixture set: %v", err)
	}
	return result
}

func fixturePDUType(name string) gosnmp.Asn1BER {
	switch name {
	case "Gauge32":
		return gosnmp.Gauge32
	case "Integer":
		return gosnmp.Integer
	case "OctetString":
		return gosnmp.OctetString
	case "NoSuchObject":
		return gosnmp.NoSuchObject
	default:
		return gosnmp.Null
	}
}
