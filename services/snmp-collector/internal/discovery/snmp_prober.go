package discovery

import (
	"context"
	"fmt"
	"strings"

	"github.com/gosnmp/gosnmp"
)

// SNMPProber performs read-only SNMPv2c identity probes with gosnmp.
type SNMPProber struct{}

// NewSNMPProber returns the default discovery prober.
func NewSNMPProber() *SNMPProber {
	return &SNMPProber{}
}

// Probe GETs sysObjectID, sysName, and sysDescr. It never uses ICMP or writes.
func (*SNMPProber) Probe(ctx context.Context, request ProbeRequest) (Identity, error) {
	if err := ctx.Err(); err != nil {
		return Identity{}, err
	}
	if !request.IP.IsValid() {
		return Identity{}, fmt.Errorf("invalid probe IP")
	}
	community := strings.TrimSpace(request.Community)
	if community == "" {
		return Identity{}, fmt.Errorf("SNMP community is required")
	}

	params := &gosnmp.GoSNMP{
		Target:    request.IP.String(),
		Port:      161,
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   request.Timeout,
		Retries:   request.Retries,
		MaxOids:   gosnmp.MaxOids,
	}
	if err := params.Connect(); err != nil {
		return Identity{}, fmt.Errorf("snmp connect: %w", err)
	}
	defer params.Conn.Close()

	if err := ctx.Err(); err != nil {
		return Identity{}, err
	}
	pkt, err := params.Get([]string{request.OIDs[0], request.OIDs[1], request.OIDs[2]})
	if err != nil {
		return Identity{}, fmt.Errorf("snmp get: %w", err)
	}
	if len(pkt.Variables) != 3 {
		return Identity{}, fmt.Errorf("identity: got %d variables, want 3", len(pkt.Variables))
	}

	objectID, err := pduText(pkt.Variables[0])
	if err != nil {
		return Identity{}, fmt.Errorf("sysObjectID: %w", err)
	}
	name, err := pduText(pkt.Variables[1])
	if err != nil {
		return Identity{}, fmt.Errorf("sysName: %w", err)
	}
	descr, err := pduText(pkt.Variables[2])
	if err != nil {
		return Identity{}, fmt.Errorf("sysDescr: %w", err)
	}
	return Identity{
		SysObjectID: objectID,
		SysName:     name,
		SysDescr:    descr,
	}, nil
}

func pduText(pdu gosnmp.SnmpPDU) (string, error) {
	switch pdu.Type {
	case gosnmp.NoSuchObject, gosnmp.NoSuchInstance, gosnmp.Null, gosnmp.EndOfMibView:
		return "", fmt.Errorf("OID %s unavailable (type %s)", pdu.Name, pdu.Type)
	}
	switch value := pdu.Value.(type) {
	case string:
		return value, nil
	case []byte:
		return string(value), nil
	default:
		return "", fmt.Errorf("OID %s unexpected type %T", pdu.Name, pdu.Value)
	}
}
