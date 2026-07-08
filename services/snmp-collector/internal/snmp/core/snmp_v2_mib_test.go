package core

import (
	"testing"

	"github.com/gosnmp/gosnmp"
)

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
