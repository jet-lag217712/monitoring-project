package core

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"
)

// IF-MIB / ifXTable OID prefixes used for interface inventory and counters.
const (
	OIDIfIndex       = "1.3.6.1.2.1.2.2.1.1"
	OIDIfDescr       = "1.3.6.1.2.1.2.2.1.2"
	OIDIfType        = "1.3.6.1.2.1.2.2.1.3"
	OIDIfSpeed       = "1.3.6.1.2.1.2.2.1.5"
	OIDIfAdminStatus = "1.3.6.1.2.1.2.2.1.7"
	OIDIfOperStatus  = "1.3.6.1.2.1.2.2.1.8"
	OIDIfLastChange  = "1.3.6.1.2.1.2.2.1.9"
	OIDIfInErrors    = "1.3.6.1.2.1.2.2.1.14"
	OIDIfOutErrors   = "1.3.6.1.2.1.2.2.1.20"
	OIDIfName        = "1.3.6.1.2.1.31.1.1.1.1"
	OIDIfHCInOctets  = "1.3.6.1.2.1.31.1.1.1.6"
	OIDIfHCOutOctets = "1.3.6.1.2.1.31.1.1.1.10"
	OIDIfHighSpeed   = "1.3.6.1.2.1.31.1.1.1.15"
	OIDIfAlias       = "1.3.6.1.2.1.31.1.1.1.18"
	// 32-bit fallbacks when ifHC* counters are unavailable on older devices.
	OIDIfInOctets  = "1.3.6.1.2.1.2.2.1.10"
	OIDIfOutOctets = "1.3.6.1.2.1.2.2.1.16"
)

// InterfaceReading holds core IF-MIB inventory and counters for one ifIndex.
type InterfaceReading struct {
	IfIndex           int
	IfDescr           string
	IfName            string
	IfAlias           string
	IfType            int
	IfTypeName        string
	SpeedBPS          uint64
	AdminStatus       string
	OperStatus        string
	LastChangeSeconds float64
	InOctets          uint64
	OutOctets         uint64
	InErrors          uint64
	OutErrors         uint64
	HasCounters       bool
}

// Walker is the SNMP walk surface used by interface polling.
type Walker interface {
	Walk(ctx context.Context, rootOID string, walkFn gosnmp.WalkFunc) error
}

// InterfacePollWalkBudget is the maximum SNMP walks PollInterfaces performs.
const InterfacePollWalkBudget = 16

// PollInterfaces walks IF-MIB / ifXTable columns and returns every interface.
// It prefers 64-bit ifHC* octet counters and falls back to 32-bit counters.
func PollInterfaces(ctx context.Context, client Walker) ([]InterfaceReading, error) {
	indexes, err := walkIndexes(ctx, client, OIDIfIndex)
	if err != nil {
		return nil, fmt.Errorf("ifIndex: %w", err)
	}
	if len(indexes) == 0 {
		return nil, nil
	}

	descrs, err := walkStrings(ctx, client, OIDIfDescr)
	if err != nil {
		return nil, fmt.Errorf("ifDescr: %w", err)
	}
	names, err := walkStrings(ctx, client, OIDIfName)
	if err != nil {
		return nil, fmt.Errorf("ifName: %w", err)
	}
	aliases, err := walkStrings(ctx, client, OIDIfAlias)
	if err != nil {
		return nil, fmt.Errorf("ifAlias: %w", err)
	}
	types, err := walkCounters(ctx, client, OIDIfType)
	if err != nil {
		return nil, fmt.Errorf("ifType: %w", err)
	}
	speeds, err := walkCounters(ctx, client, OIDIfSpeed)
	if err != nil {
		return nil, fmt.Errorf("ifSpeed: %w", err)
	}
	highSpeeds, err := walkCounters(ctx, client, OIDIfHighSpeed)
	if err != nil {
		return nil, fmt.Errorf("ifHighSpeed: %w", err)
	}
	adminStatuses, err := walkCounters(ctx, client, OIDIfAdminStatus)
	if err != nil {
		return nil, fmt.Errorf("ifAdminStatus: %w", err)
	}
	operStatuses, err := walkCounters(ctx, client, OIDIfOperStatus)
	if err != nil {
		return nil, fmt.Errorf("ifOperStatus: %w", err)
	}
	lastChanges, err := walkCounters(ctx, client, OIDIfLastChange)
	if err != nil {
		return nil, fmt.Errorf("ifLastChange: %w", err)
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
			if in32 != nil && out32 != nil {
				inOctets, okIn = in32[idx]
				outOctets, okOut = out32[idx]
			}
		}

		speed := speeds[idx]
		if highSpeeds[idx] > 0 {
			speed = highSpeeds[idx] * 1_000_000
		}
		ifType := int(types[idx])
		readings = append(readings, InterfaceReading{
			IfIndex:           idx,
			IfDescr:           descrs[idx],
			IfName:            names[idx],
			IfAlias:           aliases[idx],
			IfType:            ifType,
			IfTypeName:        InterfaceTypeName(ifType),
			SpeedBPS:          speed,
			AdminStatus:       InterfaceStatusName(int(adminStatuses[idx])),
			OperStatus:        InterfaceStatusName(int(operStatuses[idx])),
			LastChangeSeconds: float64(lastChanges[idx]) / 100,
			InOctets:          inOctets,
			OutOctets:         outOctets,
			InErrors:          inErr[idx],
			OutErrors:         outErr[idx],
			HasCounters:       okIn && okOut,
		})
	}

	sort.Slice(readings, func(i, j int) bool {
		return readings[i].IfIndex < readings[j].IfIndex
	})
	return readings, nil
}

// InterfaceTypeName returns the canonical config name for an IANA ifType.
func InterfaceTypeName(ifType int) string {
	names := map[int]string{
		1:   "other",
		6:   "ethernetcsmacd",
		15:  "fddi",
		23:  "ppp",
		24:  "softwareloopback",
		32:  "framerelay",
		37:  "atm",
		53:  "propvirtual",
		71:  "ieee80211",
		131: "tunnel",
		135: "l2vlan",
		161: "ieee8023adlag",
		209: "bridge",
	}
	if name, ok := names[ifType]; ok {
		return name
	}
	return strconv.Itoa(ifType)
}

// InterfaceStatusName returns an IF-MIB status enum name.
func InterfaceStatusName(status int) string {
	switch status {
	case 1:
		return "up"
	case 2:
		return "down"
	case 3:
		return "testing"
	default:
		return "unknown"
	}
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

func walkStrings(ctx context.Context, client Walker, rootOID string) (map[int]string, error) {
	out := make(map[int]string)
	err := client.Walk(ctx, rootOID, func(pdu gosnmp.SnmpPDU) error {
		idx, err := oidIndex(rootOID, pdu.Name)
		if err != nil {
			return err
		}
		value, err := pduToString(pdu)
		if err != nil {
			switch pdu.Type {
			case gosnmp.NoSuchObject, gosnmp.NoSuchInstance, gosnmp.Null, gosnmp.EndOfMibView:
				return nil
			default:
				return err
			}
		}
		out[idx] = value
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
