package core

import (
	"context"
	"fmt"

	"github.com/gosnmp/gosnmp"
)

// OIDSysUpTime is SNMPv2-MIB::sysUpTime.0 (TimeTicks, hundredths of a second).
const OIDSysUpTime = "1.3.6.1.2.1.1.3.0"

// Getter is the SNMP GET surface used by device polling.
type Getter interface {
	Get(ctx context.Context, oids []string) (*gosnmp.SnmpPacket, error)
}

// PollDevice reads sysUpTime and returns uptime in seconds.
func PollDevice(ctx context.Context, client Getter) (float64, error) {
	pkt, err := client.Get(ctx, []string{OIDSysUpTime})
	if err != nil {
		return 0, err
	}
	if len(pkt.Variables) == 0 {
		return 0, fmt.Errorf("sysUpTime: empty response")
	}
	return TicksToSeconds(pkt.Variables[0])
}

// TicksToSeconds converts an SNMP TimeTicks PDU value to seconds.
func TicksToSeconds(pdu gosnmp.SnmpPDU) (float64, error) {
	ticks, err := pduToUint64(pdu)
	if err != nil {
		return 0, fmt.Errorf("sysUpTime: %w", err)
	}
	return float64(ticks) / 100.0, nil
}

func pduToUint64(pdu gosnmp.SnmpPDU) (uint64, error) {
	switch pdu.Type {
	case gosnmp.NoSuchObject, gosnmp.NoSuchInstance, gosnmp.Null, gosnmp.EndOfMibView:
		return 0, fmt.Errorf("OID %s unavailable (type %s)", pdu.Name, pdu.Type)
	}

	switch v := pdu.Value.(type) {
	case uint:
		return uint64(v), nil
	case uint32:
		return uint64(v), nil
	case uint64:
		return v, nil
	case int:
		if v < 0 {
			return 0, fmt.Errorf("OID %s negative value %d", pdu.Name, v)
		}
		return uint64(v), nil
	case int32:
		if v < 0 {
			return 0, fmt.Errorf("OID %s negative value %d", pdu.Name, v)
		}
		return uint64(v), nil
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("OID %s negative value %d", pdu.Name, v)
		}
		return uint64(v), nil
	default:
		return 0, fmt.Errorf("OID %s unexpected type %T", pdu.Name, pdu.Value)
	}
}
