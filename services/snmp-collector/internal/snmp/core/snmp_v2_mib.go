package core

import (
	"context"
	"fmt"

	"github.com/gosnmp/gosnmp"
)

// SNMPv2-MIB system scalar OIDs.
const (
	OIDSysDescr    = "1.3.6.1.2.1.1.1.0"
	OIDSysObjectID = "1.3.6.1.2.1.1.2.0"
	OIDSysUpTime   = "1.3.6.1.2.1.1.3.0"
	OIDSysName     = "1.3.6.1.2.1.1.5.0"
)

// DeviceIdentity is the core identity returned for every successful poll.
type DeviceIdentity struct {
	SysObjectID   string
	SysName       string
	SysDescr      string
	UptimeSeconds float64
}

// Getter is the SNMP GET surface used by device polling.
type Getter interface {
	Get(ctx context.Context, oids []string) (*gosnmp.SnmpPacket, error)
}

// PollDevice reads the standard SNMPv2-MIB identity scalars.
func PollDevice(ctx context.Context, client Getter) (DeviceIdentity, error) {
	pkt, err := client.Get(ctx, []string{OIDSysObjectID, OIDSysName, OIDSysDescr, OIDSysUpTime})
	if err != nil {
		return DeviceIdentity{}, err
	}
	if len(pkt.Variables) != 4 {
		return DeviceIdentity{}, fmt.Errorf("identity: got %d variables, want 4", len(pkt.Variables))
	}

	objectID, err := pduToOID(pkt.Variables[0])
	if err != nil {
		return DeviceIdentity{}, fmt.Errorf("sysObjectID: %w", err)
	}
	name, err := pduToString(pkt.Variables[1])
	if err != nil {
		return DeviceIdentity{}, fmt.Errorf("sysName: %w", err)
	}
	descr, err := pduToString(pkt.Variables[2])
	if err != nil {
		return DeviceIdentity{}, fmt.Errorf("sysDescr: %w", err)
	}
	uptime, err := TicksToSeconds(pkt.Variables[3])
	if err != nil {
		return DeviceIdentity{}, err
	}
	return DeviceIdentity{
		SysObjectID:   normalizeOID(objectID),
		SysName:       name,
		SysDescr:      descr,
		UptimeSeconds: uptime,
	}, nil
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

func pduToString(pdu gosnmp.SnmpPDU) (string, error) {
	if err := unavailablePDU(pdu); err != nil {
		return "", err
	}
	switch v := pdu.Value.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		return "", fmt.Errorf("OID %s unexpected type %T", pdu.Name, pdu.Value)
	}
}

func pduToOID(pdu gosnmp.SnmpPDU) (string, error) {
	if err := unavailablePDU(pdu); err != nil {
		return "", err
	}
	switch v := pdu.Value.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		return "", fmt.Errorf("OID %s unexpected type %T", pdu.Name, pdu.Value)
	}
}

func unavailablePDU(pdu gosnmp.SnmpPDU) error {
	switch pdu.Type {
	case gosnmp.NoSuchObject, gosnmp.NoSuchInstance, gosnmp.Null, gosnmp.EndOfMibView:
		return fmt.Errorf("OID %s unavailable (type %s)", pdu.Name, pdu.Type)
	default:
		return nil
	}
}

func normalizeOID(oid string) string {
	for len(oid) > 0 && oid[0] == '.' {
		oid = oid[1:]
	}
	return oid
}
