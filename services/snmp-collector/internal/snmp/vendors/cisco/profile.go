// Package cisco collects Cisco-specific device readings.
package cisco

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"

	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/profile"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/readings"
)

const (
	oidCPU5Min          = "1.3.6.1.4.1.9.9.109.1.1.1.1.8"
	oidMemoryPoolUsed   = "1.3.6.1.4.1.9.9.48.1.1.1.5"
	oidMemoryPoolFree   = "1.3.6.1.4.1.9.9.48.1.1.1.6"
	oidTemperatureDescr = "1.3.6.1.4.1.9.9.13.1.3.1.2"
	oidTemperatureValue = "1.3.6.1.4.1.9.9.13.1.3.1.3"
	oidTemperatureState = "1.3.6.1.4.1.9.9.13.1.3.1.6"
	oidSupplyDescr      = "1.3.6.1.4.1.9.9.13.1.5.1.2"
	oidSupplyState      = "1.3.6.1.4.1.9.9.13.1.5.1.3"
)

var capabilities = readings.CapabilityCPU |
	readings.CapabilityMemory |
	readings.CapabilityTemperature |
	readings.CapabilityPower

// Profile implements Cisco IOS and IOS-XE enrichment.
type Profile struct{}

// New returns a Cisco profile.
func New() *Profile {
	return &Profile{}
}

// Name returns the stable profile identifier.
func (*Profile) Name() string {
	return "cisco"
}

// Capabilities returns the profile's static capability declaration.
func (*Profile) Capabilities() readings.Capability {
	return capabilities
}

// ExactObjectIDs returns model identifiers with fixture-tested mappings.
func (*Profile) ExactObjectIDs() []string {
	return []string{"1.3.6.1.4.1.9.1.1745"}
}

// ObjectIDPrefixes returns model-family identifiers with shared mappings.
func (*Profile) ObjectIDPrefixes() []string {
	return nil
}

// GenericVendorPrefix returns Cisco's IANA enterprise OID.
func (*Profile) GenericVendorPrefix() string {
	return "1.3.6.1.4.1.9"
}

// Collect reads supported Cisco MIB tables and preserves any valid partial result.
func (p *Profile) Collect(ctx context.Context, client profile.Client) (readings.VendorReadings, error) {
	result := readings.VendorReadings{
		Profile:      p.Name(),
		Capabilities: p.Capabilities(),
	}

	cpu := make(map[int]float64)
	memoryUsed := make(map[int]float64)
	memoryFree := make(map[int]float64)
	temperatures := make(map[int]*component)
	supplies := make(map[int]*component)

	walks := []struct {
		root string
		fn   func(gosnmp.SnmpPDU) error
	}{
		{oidCPU5Min, numericColumn(oidCPU5Min, cpu, percentValue)},
		{oidMemoryPoolUsed, numericColumn(oidMemoryPoolUsed, memoryUsed, nonNegativeValue)},
		{oidMemoryPoolFree, numericColumn(oidMemoryPoolFree, memoryFree, nonNegativeValue)},
		{oidTemperatureDescr, stringColumn(oidTemperatureDescr, temperatures)},
		{oidTemperatureValue, componentValueColumn(oidTemperatureValue, temperatures)},
		{oidTemperatureState, componentStatusColumn(oidTemperatureState, temperatures, ciscoEnvironmentStatus)},
		{oidSupplyDescr, stringColumn(oidSupplyDescr, supplies)},
		{oidSupplyState, componentStatusColumn(oidSupplyState, supplies, ciscoEnvironmentStatus)},
	}

	var collectionErrors []error
	for _, walk := range walks {
		if err := client.Walk(ctx, walk.root, walk.fn); err != nil {
			collectionErrors = append(collectionErrors, fmt.Errorf("walk %s: %w", walk.root, err))
			if ctx.Err() != nil {
				break
			}
		}
	}

	if value, ok := average(cpu); ok {
		result.CPU = &readings.ScalarReading{Value: value, SourceOID: oidCPU5Min}
	}
	if value, ok := memoryUtilization(memoryUsed, memoryFree); ok {
		result.Memory = &readings.ScalarReading{Value: value, SourceOID: oidMemoryPoolUsed}
	}
	result.Temperatures = temperatureReadings(temperatures)
	result.Power = powerReadings(supplies)

	return result, errors.Join(collectionErrors...)
}

type component struct {
	index  int
	name   string
	value  *float64
	status string
	source string
}

func numericColumn(root string, values map[int]float64, validate func(float64) error) gosnmp.WalkFunc {
	return func(pdu gosnmp.SnmpPDU) error {
		if unavailable(pdu.Type) {
			return nil
		}
		index, err := columnIndex(root, pdu.Name)
		if err != nil {
			return err
		}
		value, err := number(pdu.Value)
		if err != nil {
			return fmt.Errorf("OID %s: %w", pdu.Name, err)
		}
		if err := validate(value); err != nil {
			return fmt.Errorf("OID %s: %w", pdu.Name, err)
		}
		values[index] = value
		return nil
	}
}

func stringColumn(root string, components map[int]*component) gosnmp.WalkFunc {
	return func(pdu gosnmp.SnmpPDU) error {
		if unavailable(pdu.Type) {
			return nil
		}
		index, err := columnIndex(root, pdu.Name)
		if err != nil {
			return err
		}
		value, err := text(pdu.Value)
		if err != nil {
			return fmt.Errorf("OID %s: %w", pdu.Name, err)
		}
		entry := componentAt(components, index)
		entry.name = value
		return nil
	}
}

func componentValueColumn(root string, components map[int]*component) gosnmp.WalkFunc {
	return func(pdu gosnmp.SnmpPDU) error {
		if unavailable(pdu.Type) {
			return nil
		}
		index, err := columnIndex(root, pdu.Name)
		if err != nil {
			return err
		}
		value, err := number(pdu.Value)
		if err != nil {
			return fmt.Errorf("OID %s: %w", pdu.Name, err)
		}
		if err := nonNegativeValue(value); err != nil {
			return fmt.Errorf("OID %s: %w", pdu.Name, err)
		}
		entry := componentAt(components, index)
		entry.value = &value
		entry.source = root
		return nil
	}
}

func componentStatusColumn(root string, components map[int]*component, status func(int64) string) gosnmp.WalkFunc {
	return func(pdu gosnmp.SnmpPDU) error {
		if unavailable(pdu.Type) {
			return nil
		}
		index, err := columnIndex(root, pdu.Name)
		if err != nil {
			return err
		}
		value, err := integer(pdu.Value)
		if err != nil {
			return fmt.Errorf("OID %s: %w", pdu.Name, err)
		}
		entry := componentAt(components, index)
		entry.status = status(value)
		if entry.source == "" {
			entry.source = root
		}
		return nil
	}
}

func componentAt(components map[int]*component, index int) *component {
	if components[index] == nil {
		components[index] = &component{index: index}
	}
	return components[index]
}

func average(values map[int]float64) (float64, bool) {
	if len(values) == 0 {
		return 0, false
	}
	var total float64
	for _, value := range values {
		total += value
	}
	return total / float64(len(values)), true
}

func memoryUtilization(used, free map[int]float64) (float64, bool) {
	var totalUsed, totalFree float64
	complete := 0
	for index, usedValue := range used {
		freeValue, ok := free[index]
		if !ok {
			continue
		}
		totalUsed += usedValue
		totalFree += freeValue
		complete++
	}
	total := totalUsed + totalFree
	if complete == 0 || total <= 0 {
		return 0, false
	}
	return totalUsed / total * 100, true
}

func temperatureReadings(components map[int]*component) []readings.ComponentReading {
	return sortedComponents(components, "celsius")
}

func powerReadings(components map[int]*component) []readings.ComponentReading {
	return sortedComponents(components, "")
}

func sortedComponents(components map[int]*component, unit string) []readings.ComponentReading {
	indexes := make([]int, 0, len(components))
	for index, entry := range components {
		if entry.name == "" && entry.value == nil && entry.status == "" {
			continue
		}
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	result := make([]readings.ComponentReading, 0, len(indexes))
	for _, index := range indexes {
		entry := components[index]
		entryUnit := unit
		if entry.value == nil {
			entryUnit = ""
		}
		result = append(result, readings.ComponentReading{
			Index:     entry.index,
			Name:      entry.name,
			Value:     entry.value,
			Unit:      entryUnit,
			Status:    entry.status,
			SourceOID: entry.source,
		})
	}
	return result
}

func columnIndex(root, oid string) (int, error) {
	normalizedRoot := strings.TrimLeft(root, ".")
	normalizedOID := strings.TrimLeft(oid, ".")
	prefix := normalizedRoot + "."
	if !strings.HasPrefix(normalizedOID, prefix) {
		return 0, fmt.Errorf("OID %s is outside subtree %s", oid, root)
	}
	indexText := strings.TrimPrefix(normalizedOID, prefix)
	if strings.Contains(indexText, ".") {
		return 0, fmt.Errorf("OID %s has unsupported composite index", oid)
	}
	index, err := strconv.Atoi(indexText)
	if err != nil || index < 0 {
		return 0, fmt.Errorf("OID %s has invalid index", oid)
	}
	return index, nil
}

func number(value any) (float64, error) {
	switch value := value.(type) {
	case int:
		return float64(value), nil
	case int32:
		return float64(value), nil
	case int64:
		return float64(value), nil
	case uint:
		return float64(value), nil
	case uint32:
		return float64(value), nil
	case uint64:
		return float64(value), nil
	case float32:
		return float64(value), nil
	case float64:
		return value, nil
	default:
		return 0, fmt.Errorf("unexpected numeric type %T", value)
	}
}

func integer(value any) (int64, error) {
	numberValue, err := number(value)
	if err != nil {
		return 0, err
	}
	integerValue := int64(numberValue)
	if float64(integerValue) != numberValue {
		return 0, fmt.Errorf("non-integer value %v", numberValue)
	}
	return integerValue, nil
}

func text(value any) (string, error) {
	switch value := value.(type) {
	case string:
		return value, nil
	case []byte:
		return string(value), nil
	default:
		return "", fmt.Errorf("unexpected string type %T", value)
	}
}

func percentValue(value float64) error {
	if value < 0 || value > 100 {
		return fmt.Errorf("percentage %v outside 0..100", value)
	}
	return nil
}

func nonNegativeValue(value float64) error {
	if value < 0 {
		return fmt.Errorf("negative value %v", value)
	}
	return nil
}

func unavailable(pduType gosnmp.Asn1BER) bool {
	switch pduType {
	case gosnmp.NoSuchObject, gosnmp.NoSuchInstance, gosnmp.Null, gosnmp.EndOfMibView:
		return true
	default:
		return false
	}
}

func ciscoEnvironmentStatus(value int64) string {
	// Map CISCO-ENVMON ciscoEnvMonState onto the v2 component status enum.
	switch value {
	case 1:
		return "ok"
	case 2:
		return "warning"
	case 3:
		return "critical"
	case 5:
		return "not_present"
	default:
		return "unknown"
	}
}
