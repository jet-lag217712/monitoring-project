package core

import (
	"context"
	"testing"

	"github.com/gosnmp/gosnmp"
)

type fakeGetter struct {
	packet *gosnmp.SnmpPacket
	err    error
}

func (f fakeGetter) Get(_ context.Context, _ []string) (*gosnmp.SnmpPacket, error) {
	return f.packet, f.err
}

func TestPollDeviceIdentity(t *testing.T) {
	t.Parallel()

	identity, err := PollDevice(context.Background(), fakeGetter{packet: &gosnmp.SnmpPacket{
		Variables: []gosnmp.SnmpPDU{
			{Name: OIDSysObjectID, Type: gosnmp.ObjectIdentifier, Value: ".1.3.6.1.4.1.9.1.1745"},
			{Name: OIDSysName, Type: gosnmp.OctetString, Value: []byte("switch-01")},
			{Name: OIDSysDescr, Type: gosnmp.OctetString, Value: "Synthetic IOS"},
			{Name: OIDSysUpTime, Type: gosnmp.TimeTicks, Value: uint32(12345)},
		},
	}})
	if err != nil {
		t.Fatalf("PollDevice: %v", err)
	}
	if identity.SysObjectID != "1.3.6.1.4.1.9.1.1745" ||
		identity.SysName != "switch-01" ||
		identity.SysDescr != "Synthetic IOS" ||
		identity.UptimeSeconds != 123.45 {
		t.Fatalf("identity=%#v", identity)
	}
}

func TestPollDeviceIdentityRejectsUnavailableRequiredScalar(t *testing.T) {
	t.Parallel()

	_, err := PollDevice(context.Background(), fakeGetter{packet: &gosnmp.SnmpPacket{
		Variables: []gosnmp.SnmpPDU{
			{Name: OIDSysObjectID, Type: gosnmp.NoSuchObject},
			{Name: OIDSysName, Type: gosnmp.OctetString, Value: "switch-01"},
			{Name: OIDSysDescr, Type: gosnmp.OctetString, Value: "Synthetic"},
			{Name: OIDSysUpTime, Type: gosnmp.TimeTicks, Value: uint32(100)},
		},
	}})
	if err == nil {
		t.Fatal("expected unavailable sysObjectID error")
	}
}

func TestTicksToSeconds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pdu     gosnmp.SnmpPDU
		want    float64
		wantErr bool
	}{
		{
			name: "uint32 ticks",
			pdu:  gosnmp.SnmpPDU{Name: OIDSysUpTime, Type: gosnmp.TimeTicks, Value: uint32(123456)},
			want: 1234.56,
		},
		{
			name: "uint64 ticks",
			pdu:  gosnmp.SnmpPDU{Name: OIDSysUpTime, Type: gosnmp.TimeTicks, Value: uint64(100)},
			want: 1.0,
		},
		{
			name: "zero",
			pdu:  gosnmp.SnmpPDU{Name: OIDSysUpTime, Type: gosnmp.TimeTicks, Value: uint32(0)},
			want: 0,
		},
		{
			name:    "no such object",
			pdu:     gosnmp.SnmpPDU{Name: OIDSysUpTime, Type: gosnmp.NoSuchObject, Value: nil},
			wantErr: true,
		},
		{
			name:    "unexpected type",
			pdu:     gosnmp.SnmpPDU{Name: OIDSysUpTime, Type: gosnmp.OctetString, Value: []byte("bad")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := TicksToSeconds(tt.pdu)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
