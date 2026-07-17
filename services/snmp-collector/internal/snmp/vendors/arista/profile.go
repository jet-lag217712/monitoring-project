// Package arista collects Arista EOS-specific device readings.
package arista

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"

	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/profile"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/readings"
)

const (
	oidProcessorLoad         = "1.3.6.1.2.1.25.3.3.1.2"
	oidStorageType           = "1.3.6.1.2.1.25.2.3.1.2"
	oidStorageAllocUnits     = "1.3.6.1.2.1.25.2.3.1.4"
	oidStorageSize           = "1.3.6.1.2.1.25.2.3.1.5"
	oidStorageUsed           = "1.3.6.1.2.1.25.2.3.1.6"
	oidPhysicalDescr         = "1.3.6.1.2.1.47.1.1.1.1.2"
	oidPhysicalClass         = "1.3.6.1.2.1.47.1.1.1.1.5"
	oidEntityStateOper       = "1.3.6.1.2.1.131.1.1.1.3"
	oidSensorType            = "1.3.6.1.2.1.99.1.1.1.1"
	oidSensorScale           = "1.3.6.1.2.1.99.1.1.1.2"
	oidSensorPrecision       = "1.3.6.1.2.1.99.1.1.1.3"
	oidSensorValue           = "1.3.6.1.2.1.99.1.1.1.4"
	oidSensorOperStatus      = "1.3.6.1.2.1.99.1.1.1.5"
	oidHostResourcesFixedRAM = "1.3.6.1.2.1.25.2.1.2"

	entityClassPowerSupply = 6
	sensorTypeVoltsAC      = 3
	sensorTypeVoltsDC      = 4
	sensorTypeAmperes      = 5
	sensorTypeWatts        = 6
	sensorTypeCelsius      = 8
)

var capabilities = readings.CapabilityCPU |
	readings.CapabilityMemory |
	readings.CapabilityTemperature |
	readings.CapabilityPower

// Profile implements Arista EOS enrichment using supported standard MIBs.
type Profile struct{}

// New returns an Arista profile.
func New() *Profile {
	return &Profile{}
}

// Name returns the stable profile identifier.
func (*Profile) Name() string {
	return "arista"
}

// Capabilities returns the profile's static capability declaration.
func (*Profile) Capabilities() readings.Capability {
	return capabilities
}

// ExactObjectIDs returns model identifiers with fixture-tested mappings.
func (*Profile) ExactObjectIDs() []string {
	return []string{"1.3.6.1.4.1.30065.1.3011.7050.3282.52"}
}

// ObjectIDPrefixes returns model-family identifiers with shared EOS mappings.
func (*Profile) ObjectIDPrefixes() []string {
	return []string{"1.3.6.1.4.1.30065.1.3011.7050"}
}

// GenericVendorPrefix returns Arista's IANA enterprise OID.
func (*Profile) GenericVendorPrefix() string {
	return "1.3.6.1.4.1.30065"
}

// Collect reads EOS-supported HOST-RESOURCES and ENTITY sensor tables.
func (p *Profile) Collect(ctx context.Context, client profile.Client) (readings.VendorReadings, error) {
	result := readings.VendorReadings{
		Profile:      p.Name(),
		Capabilities: p.Capabilities(),
	}

	cpu := make(map[int]float64)
	storageTypes := make(map[int]string)
	allocationUnits := make(map[int]float64)
	storageSizes := make(map[int]float64)
	storageUsed := make(map[int]float64)
	physical := make(map[int]*physicalEntity)
	sensors := make(map[int]*sensor)

	walks := []struct {
		root string
		fn   func(gosnmp.SnmpPDU) error
	}{
		{oidProcessorLoad, numericColumn(oidProcessorLoad, cpu, percentValue)},
		{oidStorageType, oidColumn(oidStorageType, storageTypes)},
		{oidStorageAllocUnits, numericColumn(oidStorageAllocUnits, allocationUnits, positiveValue)},
		{oidStorageSize, numericColumn(oidStorageSize, storageSizes, nonNegativeValue)},
		{oidStorageUsed, numericColumn(oidStorageUsed, storageUsed, nonNegativeValue)},
		{oidPhysicalDescr, physicalStringColumn(oidPhysicalDescr, physical)},
		{oidPhysicalClass, physicalIntegerColumn(oidPhysicalClass, physical, func(entry *physicalEntity, value int64) {
			entry.class = value
		})},
		{oidEntityStateOper, physicalIntegerColumn(oidEntityStateOper, physical, func(entry *physicalEntity, value int64) {
			entry.status = entityOperStatus(value)
		})},
		{oidSensorType, sensorIntegerColumn(oidSensorType, sensors, func(entry *sensor, value int64) {
			entry.sensorType = value
			entry.hasType = true
		})},
		{oidSensorScale, sensorIntegerColumn(oidSensorScale, sensors, func(entry *sensor, value int64) {
			entry.scale = value
			entry.hasScale = true
		})},
		{oidSensorPrecision, sensorIntegerColumn(oidSensorPrecision, sensors, func(entry *sensor, value int64) {
			entry.precision = value
			entry.hasPrecision = true
		})},
		{oidSensorValue, sensorIntegerColumn(oidSensorValue, sensors, func(entry *sensor, value int64) {
			entry.value = value
			entry.hasValue = true
		})},
		{oidSensorOperStatus, sensorIntegerColumn(oidSensorOperStatus, sensors, func(entry *sensor, value int64) {
			entry.status = sensorOperStatus(value)
		})},
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
		result.CPU = &readings.ScalarReading{Value: value, SourceOID: oidProcessorLoad}
	}
	if value, ok := memoryUtilization(storageTypes, allocationUnits, storageSizes, storageUsed); ok {
		result.Memory = &readings.ScalarReading{Value: value, SourceOID: oidStorageUsed}
	}
	result.Temperatures, result.Power = componentReadings(physical, sensors)

	return result, errors.Join(collectionErrors...)
}

type physicalEntity struct {
	index  int
	name   string
	class  int64
	status string
}

type sensor struct {
	index        int
	sensorType   int64
	hasType      bool
	scale        int64
	hasScale     bool
	precision    int64
	hasPrecision bool
	value        int64
	hasValue     bool
	status       string
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

func oidColumn(root string, values map[int]string) gosnmp.WalkFunc {
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
		values[index] = strings.TrimLeft(value, ".")
		return nil
	}
}

func physicalStringColumn(root string, entries map[int]*physicalEntity) gosnmp.WalkFunc {
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
		physicalAt(entries, index).name = value
		return nil
	}
}

func physicalIntegerColumn(root string, entries map[int]*physicalEntity, assign func(*physicalEntity, int64)) gosnmp.WalkFunc {
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
		assign(physicalAt(entries, index), value)
		return nil
	}
}

func sensorIntegerColumn(root string, entries map[int]*sensor, assign func(*sensor, int64)) gosnmp.WalkFunc {
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
		assign(sensorAt(entries, index), value)
		return nil
	}
}

func physicalAt(entries map[int]*physicalEntity, index int) *physicalEntity {
	if entries[index] == nil {
		entries[index] = &physicalEntity{index: index}
	}
	return entries[index]
}

func sensorAt(entries map[int]*sensor, index int) *sensor {
	if entries[index] == nil {
		entries[index] = &sensor{index: index}
	}
	return entries[index]
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

func memoryUtilization(types map[int]string, allocationUnits, sizes, used map[int]float64) (float64, bool) {
	var usedBytes, totalBytes float64
	complete := 0
	for index, storageType := range types {
		if storageType != oidHostResourcesFixedRAM {
			continue
		}
		unit, hasUnit := allocationUnits[index]
		size, hasSize := sizes[index]
		usedSize, hasUsed := used[index]
		if !hasUnit || !hasSize || !hasUsed || usedSize > size {
			continue
		}
		usedBytes += usedSize * unit
		totalBytes += size * unit
		complete++
	}
	if complete == 0 || totalBytes <= 0 {
		return 0, false
	}
	return usedBytes / totalBytes * 100, true
}

func componentReadings(physical map[int]*physicalEntity, sensors map[int]*sensor) ([]readings.ComponentReading, []readings.ComponentReading) {
	indexes := make([]int, 0, len(sensors))
	for index := range sensors {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	var temperatures []readings.ComponentReading
	var power []readings.ComponentReading
	for _, index := range indexes {
		entry := sensors[index]
		if !entry.hasType {
			continue
		}
		unit, destination := sensorDestination(entry.sensorType)
		if destination == "" {
			continue
		}
		var value *float64
		if entry.hasValue && entry.hasScale && entry.hasPrecision {
			scaled := float64(entry.value) * math.Pow10(scaleExponent(entry.scale)-int(entry.precision))
			value = &scaled
		}
		name := ""
		if physical[index] != nil {
			name = physical[index].name
		}
		reading := readings.ComponentReading{
			Index:     index,
			Name:      name,
			Value:     value,
			Status:    entry.status,
			SourceOID: oidSensorValue,
		}
		if value != nil {
			reading.Unit = unit
		}
		if destination == "temperature" {
			temperatures = append(temperatures, reading)
		} else {
			power = append(power, reading)
		}
	}

	physicalIndexes := make([]int, 0, len(physical))
	for index, entry := range physical {
		if entry.class == entityClassPowerSupply {
			physicalIndexes = append(physicalIndexes, index)
		}
	}
	sort.Ints(physicalIndexes)
	for _, index := range physicalIndexes {
		entry := physical[index]
		power = append(power, readings.ComponentReading{
			Index:     index,
			Name:      entry.name,
			Status:    entry.status,
			SourceOID: oidEntityStateOper,
		})
	}
	return temperatures, power
}

func sensorDestination(sensorType int64) (unit, destination string) {
	switch sensorType {
	case sensorTypeCelsius:
		return "celsius", "temperature"
	case sensorTypeWatts:
		return "watts", "power"
	case sensorTypeVoltsAC:
		return "volts_ac", "power"
	case sensorTypeVoltsDC:
		return "volts_dc", "power"
	case sensorTypeAmperes:
		return "amperes", "power"
	default:
		return "", ""
	}
}

func scaleExponent(scale int64) int {
	// EntitySensorDataScale runs from yocto (-24) through units (0) to yotta (+24).
	if scale < 1 || scale > 17 {
		return 0
	}
	return int(scale-9) * 3
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

func positiveValue(value float64) error {
	if value <= 0 {
		return fmt.Errorf("non-positive value %v", value)
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

func sensorOperStatus(value int64) string {
	// Map EntitySensorStatus onto the v2 component status enum.
	switch value {
	case 1:
		return "ok"
	case 3:
		return "critical"
	default:
		return "unknown"
	}
}

func entityOperStatus(value int64) string {
	// Map EntityOperStatus onto the v2 component status enum for power supplies.
	switch value {
	case 2:
		return "ok"
	case 3:
		return "not_present"
	default:
		return "unknown"
	}
}
