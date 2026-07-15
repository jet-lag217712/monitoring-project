package core

import (
	"context"
	"testing"

	"github.com/gosnmp/gosnmp"
)

type fakeWalker struct {
	columns map[string][]gosnmp.SnmpPDU
}

func (f *fakeWalker) Walk(_ context.Context, rootOID string, walkFn gosnmp.WalkFunc) error {
	for _, pdu := range f.columns[rootOID] {
		if err := walkFn(pdu); err != nil {
			return err
		}
	}
	return nil
}

func TestParseInterfaceWalk(t *testing.T) {
	t.Parallel()

	pdus := []gosnmp.SnmpPDU{
		{Name: OIDIfHCInOctets + ".1", Type: gosnmp.Counter64, Value: uint64(100)},
		{Name: OIDIfHCInOctets + ".2", Type: gosnmp.Counter64, Value: uint64(200)},
	}
	got, err := ParseInterfaceWalk(OIDIfHCInOctets, pdus)
	if err != nil {
		t.Fatalf("ParseInterfaceWalk: %v", err)
	}
	if got[1] != 100 || got[2] != 200 {
		t.Fatalf("unexpected map: %#v", got)
	}
}

func TestPollInterfacesHC(t *testing.T) {
	t.Parallel()

	w := &fakeWalker{columns: map[string][]gosnmp.SnmpPDU{
		OIDIfIndex: {
			{Name: OIDIfIndex + ".1", Type: gosnmp.Integer, Value: 1},
			{Name: OIDIfIndex + ".2", Type: gosnmp.Integer, Value: 2},
		},
		OIDIfHCInOctets: {
			{Name: OIDIfHCInOctets + ".1", Type: gosnmp.Counter64, Value: uint64(10)},
			{Name: OIDIfHCInOctets + ".2", Type: gosnmp.Counter64, Value: uint64(20)},
		},
		OIDIfHCOutOctets: {
			{Name: OIDIfHCOutOctets + ".1", Type: gosnmp.Counter64, Value: uint64(11)},
			{Name: OIDIfHCOutOctets + ".2", Type: gosnmp.Counter64, Value: uint64(21)},
		},
		OIDIfInErrors: {
			{Name: OIDIfInErrors + ".1", Type: gosnmp.Counter32, Value: uint32(1)},
			{Name: OIDIfInErrors + ".2", Type: gosnmp.Counter32, Value: uint32(2)},
		},
		OIDIfOutErrors: {
			{Name: OIDIfOutErrors + ".1", Type: gosnmp.Counter32, Value: uint32(3)},
			{Name: OIDIfOutErrors + ".2", Type: gosnmp.Counter32, Value: uint32(4)},
		},
	}}

	readings, err := PollInterfaces(context.Background(), w)
	if err != nil {
		t.Fatalf("PollInterfaces: %v", err)
	}
	if len(readings) != 2 {
		t.Fatalf("got %d readings, want 2", len(readings))
	}
	if readings[0].IfIndex != 1 || readings[0].InOctets != 10 || readings[0].OutOctets != 11 {
		t.Fatalf("reading[0]=%#v", readings[0])
	}
	if readings[1].IfIndex != 2 || readings[1].InErrors != 2 || readings[1].OutErrors != 4 {
		t.Fatalf("reading[1]=%#v", readings[1])
	}
}

func TestPollInterfacesFallback32Bit(t *testing.T) {
	t.Parallel()

	w := &fakeWalker{columns: map[string][]gosnmp.SnmpPDU{
		OIDIfIndex: {
			{Name: OIDIfIndex + ".5", Type: gosnmp.Integer, Value: 5},
		},
		OIDIfHCInOctets:  {},
		OIDIfHCOutOctets: {},
		OIDIfInOctets: {
			{Name: OIDIfInOctets + ".5", Type: gosnmp.Counter32, Value: uint32(99)},
		},
		OIDIfOutOctets: {
			{Name: OIDIfOutOctets + ".5", Type: gosnmp.Counter32, Value: uint32(88)},
		},
		OIDIfInErrors: {
			{Name: OIDIfInErrors + ".5", Type: gosnmp.Counter32, Value: uint32(0)},
		},
		OIDIfOutErrors: {
			{Name: OIDIfOutErrors + ".5", Type: gosnmp.Counter32, Value: uint32(0)},
		},
	}}

	readings, err := PollInterfaces(context.Background(), w)
	if err != nil {
		t.Fatalf("PollInterfaces: %v", err)
	}
	if len(readings) != 1 {
		t.Fatalf("got %d readings, want 1", len(readings))
	}
	if readings[0].InOctets != 99 || readings[0].OutOctets != 88 {
		t.Fatalf("unexpected fallback values: %#v", readings[0])
	}
}
