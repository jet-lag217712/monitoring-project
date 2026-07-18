package core

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/gosnmp/gosnmp"
)

// cdpCacheAddressOID is CISCO-CDP-MIB::cdpCacheAddress (neighbor L3 address).
const cdpCacheAddressOID = "1.3.6.1.4.1.9.9.23.1.2.1.1.4"

// CDPNeighborIPs walks the CDP cache and returns unique neighbor IPv4 addresses.
func CDPNeighborIPs(ctx context.Context, client interface {
	Walk(context.Context, string, gosnmp.WalkFunc) error
}) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	var out []string
	err := client.Walk(ctx, cdpCacheAddressOID, func(pdu gosnmp.SnmpPDU) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		ip, ok := parseCDPAddress(pdu)
		if !ok || ip == "" {
			return nil
		}
		if _, exists := seen[ip]; exists {
			return nil
		}
		seen[ip] = struct{}{}
		out = append(out, ip)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// parseCDPAddress decodes CISCO-CDP-MIB NetworkAddress octets (type 1 = IPv4).
func parseCDPAddress(pdu gosnmp.SnmpPDU) (string, bool) {
	raw, ok := pdu.Value.([]byte)
	if !ok || len(raw) < 5 {
		return "", false
	}
	if raw[0] != 1 {
		return "", false
	}
	addrLen := int(raw[1])
	if addrLen != 4 || len(raw) < 2+addrLen {
		return "", false
	}
	ip := net.IP(raw[2 : 2+addrLen])
	if ip == nil || ip.IsUnspecified() {
		return "", false
	}
	return ip.String(), true
}

// ProbeCDPNeighborIPs opens a short SNMP session and returns CDP neighbor IPs.
func ProbeCDPNeighborIPs(ctx context.Context, target, community string, timeout time.Duration, retries int) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	params := &gosnmp.GoSNMP{
		Target:    target,
		Port:      161,
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   timeout,
		Retries:   retries,
		MaxOids:   gosnmp.MaxOids,
	}
	if err := params.Connect(); err != nil {
		return nil, fmt.Errorf("snmp connect: %w", err)
	}
	defer params.Conn.Close()

	walker := &gosnmpWalker{params: params}
	return CDPNeighborIPs(ctx, walker)
}

type gosnmpWalker struct {
	params *gosnmp.GoSNMP
}

func (w *gosnmpWalker) Walk(ctx context.Context, rootOID string, walkFn gosnmp.WalkFunc) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := w.params.BulkWalk(rootOID, walkFn); err != nil {
		return fmt.Errorf("snmp walk %s: %w", rootOID, err)
	}
	return nil
}
