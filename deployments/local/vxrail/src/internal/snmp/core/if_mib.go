package core

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"
)

// IF-MIB / ifXTable OID prefixes used for interface counter collection.
const (
	OIDIfIndex       = "1.3.6.1.2.1.2.2.1.1"
	OIDIfInErrors    = "1.3.6.1.2.1.2.2.1.14"
	OIDIfOutErrors   = "1.3.6.1.2.1.2.2.1.20"
	OIDIfHCInOctets  = "1.3.6.1.2.1.31.1.1.1.6"
	OIDIfHCOutOctets = "1.3.6.1.2.1.31.1.1.1.10"
	// 32-bit fallbacks when ifHC* counters are unavailable on older devices.
	OIDIfInOctets  = "1.3.6.1.2.1.2.2.1.10"
	OIDIfOutOctets = "1.3.6.1.2.1.2.2.1.16"
)

// InterfaceReading holds raw IF-MIB counters for a single ifIndex.
type InterfaceReading struct {
	IfIndex   int
	InOctets  uint64
	OutOctets uint64
	InErrors  uint64
	OutErrors uint64
}

// Walker is the SNMP walk surface used by interface polling.
type Walker interface {
	Walk(ctx context.Context, rootOID string, walkFn gosnmp.WalkFunc) error
}

// PollInterfaces walks IF-MIB / ifXTable columns and returns per-interface readings.
// Prefers 64-bit ifHC* octet counters; falls back to 32-bit ifInOctets/ifOutOctets
// when HC counters are missing. Interfaces without any octet counters are skipped.
func PollInterfaces(ctx context.Context, client Walker) ([]InterfaceReading, error) {
	indexes, err := walkIndexes(ctx, client, OIDIfIndex)
	if err != nil {
		return nil, fmt.Errorf("ifIndex: %w", err)
	}
	if len(indexes) == 0 {
		return nil, nil
	}

	hcIn, err := walkCounters(ctx, client, OIDIfHCInOctets)
	if err != nil {
		return nil, fmt.Errorf("ifHCInOctets: %w", err)
	}
	hcOut, err := walkCounters(ctx, client, OIDIfHCOutOctets)
	if err != nil {
		return nil, fmt.Errorf("ifHCOutOctets: %w", err)
	}
	inErr, err := walkCounters(ctx, client, OIDIfInErrors)
	if err != nil {
		return nil, fmt.Errorf("ifInErrors: %w", err)
	}
	outErr, err := walkCounters(ctx, client, OIDIfOutErrors)
	if err != nil {
		return nil, fmt.Errorf("ifOutErrors: %w", err)
	}

	needFallback := false
	for _, idx := range indexes {
		if _, ok := hcIn[idx]; !ok {
			needFallback = true
			break
		}
		if _, ok := hcOut[idx]; !ok {
			needFallback = true
			break
		}
	}

	var in32, out32 map[int]uint64
	if needFallback {
		in32, err = walkCounters(ctx, client, OIDIfInOctets)
		if err != nil {
			return nil, fmt.Errorf("ifInOctets: %w", err)
		}
		out32, err = walkCounters(ctx, client, OIDIfOutOctets)
		if err != nil {
			return nil, fmt.Errorf("ifOutOctets: %w", err)
		}
	}

	readings := make([]InterfaceReading, 0, len(indexes))
	for _, idx := range indexes {
		inOctets, okIn := hcIn[idx]
		outOctets, okOut := hcOut[idx]
		if !okIn || !okOut {
			if in32 == nil || out32 == nil {
				continue
			}
			inOctets, okIn = in32[idx]
			outOctets, okOut = out32[idx]
			if !okIn || !okOut {
				continue
			}
		}

		readings = append(readings, InterfaceReading{
			IfIndex:   idx,
			InOctets:  inOctets,
			OutOctets: outOctets,
			InErrors:  inErr[idx],
			OutErrors: outErr[idx],
		})
	}

	sort.Slice(readings, func(i, j int) bool {
		return readings[i].IfIndex < readings[j].IfIndex
	})
	return readings, nil
}

func walkIndexes(ctx context.Context, client Walker, rootOID string) ([]int, error) {
	var indexes []int
	err := client.Walk(ctx, rootOID, func(pdu gosnmp.SnmpPDU) error {
		idx, err := oidIndex(rootOID, pdu.Name)
		if err != nil {
			return err
		}
		indexes = append(indexes, idx)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Ints(indexes)
	return indexes, nil
}

func walkCounters(ctx context.Context, client Walker, rootOID string) (map[int]uint64, error) {
	out := make(map[int]uint64)
	err := client.Walk(ctx, rootOID, func(pdu gosnmp.SnmpPDU) error {
		idx, err := oidIndex(rootOID, pdu.Name)
		if err != nil {
			return err
		}
		v, err := pduToUint64(pdu)
		if err != nil {
			switch pdu.Type {
			case gosnmp.NoSuchObject, gosnmp.NoSuchInstance, gosnmp.Null, gosnmp.EndOfMibView:
				// Skip unavailable instances rather than failing the whole walk.
				return nil
			default:
				return err
			}
		}
		out[idx] = v
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// oidIndex extracts the trailing ifIndex from a fully-qualified OID name.
func oidIndex(rootOID, fullOID string) (int, error) {
	name := strings.TrimPrefix(fullOID, ".")
	root := strings.TrimPrefix(rootOID, ".")
	if !strings.HasPrefix(name, root+".") {
		return 0, fmt.Errorf("OID %s not under %s", fullOID, rootOID)
	}
	suffix := strings.TrimPrefix(name, root+".")
	idx, err := strconv.Atoi(suffix)
	if err != nil {
		return 0, fmt.Errorf("parse ifIndex from %s: %w", fullOID, err)
	}
	return idx, nil
}

// ParseInterfaceWalk is exported for unit tests that feed synthetic PDUs.
func ParseInterfaceWalk(rootOID string, pdus []gosnmp.SnmpPDU) (map[int]uint64, error) {
	out := make(map[int]uint64, len(pdus))
	for _, pdu := range pdus {
		idx, err := oidIndex(rootOID, pdu.Name)
		if err != nil {
			return nil, err
		}
		v, err := pduToUint64(pdu)
		if err != nil {
			continue
		}
		out[idx] = v
	}
	return out, nil
}
