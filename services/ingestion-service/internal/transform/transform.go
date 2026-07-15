package transform

import (
	"fmt"
	"time"

	"github.com/equate/ogsd/services/ingestion-service/internal/validate"
	"github.com/google/uuid"
)

// Fixed OGSD namespace for deterministic UUID v5 derivation.
var ogsdNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8") // DNS namespace as base

func init() {
	// Derive a project-specific namespace from the DNS namespace + "equate-ogsd".
	ogsdNamespace = uuid.NewSHA1(ogsdNamespace, []byte("equate-ogsd"))
}

// DeviceSample is a transformed device metric ready for persistence.
type DeviceSample struct {
	SiteUUID        uuid.UUID
	SiteName        string
	DeviceUUID      uuid.UUID
	DeviceHostname  string
	DeviceIPAddress string
	MetricName      string
	Value           float64
	CollectedAt     time.Time
}

// InterfaceSample is a transformed interface metric ready for persistence.
type InterfaceSample struct {
	SiteUUID       uuid.UUID
	SiteName       string
	DeviceUUID     uuid.UUID
	DeviceHostname string
	InterfaceUUID  uuid.UUID
	IfIndex        int
	InOctets       uint64
	OutOctets      uint64
	InErrors       uint64
	OutErrors      uint64
	CollectedAt    time.Time
}

// SiteUUID returns a deterministic UUID for a collector site_id string.
func SiteUUID(siteID string) uuid.UUID {
	return uuid.NewSHA1(ogsdNamespace, []byte("site:"+siteID))
}

// DeviceUUID returns a deterministic UUID for a site+device pair.
func DeviceUUID(siteID, deviceID string) uuid.UUID {
	return uuid.NewSHA1(ogsdNamespace, []byte("device:"+siteID+"/"+deviceID))
}

// InterfaceUUID returns a deterministic UUID for a device+ifIndex pair.
func InterfaceUUID(deviceUUID uuid.UUID, ifIndex int) uuid.UUID {
	return uuid.NewSHA1(ogsdNamespace, []byte(fmt.Sprintf("interface:%s/%d", deviceUUID.String(), ifIndex)))
}

// DeviceSampleFromValidated maps a validated device message to a store sample.
func DeviceSampleFromValidated(msg validate.DeviceMessage) DeviceSample {
	return DeviceSample{
		SiteUUID:        SiteUUID(msg.SiteID),
		SiteName:        msg.SiteID,
		DeviceUUID:      DeviceUUID(msg.SiteID, msg.DeviceID),
		DeviceHostname:  msg.DeviceID,
		DeviceIPAddress: msg.IPAddress,
		MetricName:      msg.Metric,
		Value:           msg.Value,
		CollectedAt:     msg.Timestamp,
	}
}

// InterfaceSampleFromValidated maps a validated interface message to a store sample.
func InterfaceSampleFromValidated(msg validate.InterfaceMessage) InterfaceSample {
	devUUID := DeviceUUID(msg.SiteID, msg.DeviceID)
	return InterfaceSample{
		SiteUUID:       SiteUUID(msg.SiteID),
		SiteName:       msg.SiteID,
		DeviceUUID:     devUUID,
		DeviceHostname: msg.DeviceID,
		InterfaceUUID:  InterfaceUUID(devUUID, msg.IfIndex),
		IfIndex:        msg.IfIndex,
		InOctets:       msg.InOctets,
		OutOctets:      msg.OutOctets,
		InErrors:       msg.InErrors,
		OutErrors:      msg.OutErrors,
		CollectedAt:    msg.Timestamp,
	}
}
