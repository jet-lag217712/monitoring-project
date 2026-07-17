package derive

import (
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Fixed OGSD namespace for deterministic UUID v5 derivation (must match ingestion-service).
var ogsdNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

func init() {
	ogsdNamespace = uuid.NewSHA1(ogsdNamespace, []byte("equate-ogsd"))
}

// DeviceUUID returns the deterministic device primary key for a collector site+device pair.
func DeviceUUID(siteID, deviceID string) uuid.UUID {
	return uuid.NewSHA1(ogsdNamespace, []byte("device:"+siteID+"/"+deviceID))
}

const placeholderIP = "0.0.0.0"

// Device health numeric compatibility (v2).
const (
	StatusUnknown  = 0
	StatusHealthy  = 1
	StatusWarning  = 2
	StatusCritical = 3
)

// DeviceOnline reports whether a device is considered online.
func DeviceOnline(status string, lastSeen *time.Time, now time.Time, threshold time.Duration) bool {
	if status != "online" {
		return false
	}
	if lastSeen == nil {
		return false
	}
	return now.Sub(*lastSeen) <= threshold
}

// DeviceStatusCode returns frontend numeric status from online flag: 1 healthy, 3 critical.
// Prefer HealthStatusCode when a device_health_current row exists.
func DeviceStatusCode(online bool) int {
	if online {
		return StatusHealthy
	}
	return StatusCritical
}

// HealthStatusCode maps persisted collector health state to numeric compatibility.
// Empty/unknown state strings fall back to the online-derived code.
func HealthStatusCode(state string, online bool) int {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "healthy":
		return StatusHealthy
	case "warning":
		return StatusWarning
	case "critical":
		return StatusCritical
	case "unknown":
		return StatusUnknown
	default:
		return DeviceStatusCode(online)
	}
}

// DeviceProjection holds the status fields projected for API responses.
type DeviceProjection struct {
	Status                 int
	StatusReason           string
	FailureCount           *int
	UpstreamDeviceIDs      []string
	UnavailableUpstreamIDs []string
	RootCauseDeviceIDs     []string
}

// ProjectDeviceStatus prefers health-current evidence when present.
func ProjectDeviceStatus(healthState string, healthPresent bool, reason string, failureCount int, upstream, unavailable, rootCause []string, online bool) DeviceProjection {
	out := DeviceProjection{
		UpstreamDeviceIDs:      nonNilStrings(upstream),
		UnavailableUpstreamIDs: nonNilStrings(unavailable),
		RootCauseDeviceIDs:     nonNilStrings(rootCause),
	}
	if healthPresent {
		out.Status = HealthStatusCode(healthState, online)
		out.StatusReason = reason
		fc := failureCount
		out.FailureCount = &fc
		return out
	}
	out.Status = DeviceStatusCode(online)
	return out
}

func nonNilStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

// SiteHealthCounts aggregates v2 health counts for a site.
type SiteHealthCounts struct {
	HealthyCount            int
	WarningCount            int
	CriticalCount           int
	UnknownCount            int
	DependencyImpactedCount int
}

// Accumulate updates counts from one device projection.
func (c *SiteHealthCounts) Accumulate(status int, unavailableUpstream []string) {
	switch status {
	case StatusHealthy:
		c.HealthyCount++
	case StatusWarning:
		c.WarningCount++
	case StatusCritical:
		c.CriticalCount++
	case StatusUnknown:
		c.UnknownCount++
		if len(unavailableUpstream) > 0 {
			c.DependencyImpactedCount++
		}
	}
}

// SiteStatusFromHealth returns site string status from v2 aggregates.
// alert if any direct Critical; else caution if any Warning or Unknown; else ok.
func SiteStatusFromHealth(counts SiteHealthCounts) string {
	if counts.CriticalCount > 0 {
		return "alert"
	}
	if counts.WarningCount > 0 || counts.UnknownCount > 0 {
		return "caution"
	}
	return "ok"
}

// SiteStatus returns "ok" or "alert" from device online flags (MVP fallback).
// Prefer SiteStatusFromHealth when health projections are available.
func SiteStatus(anyOffline bool) string {
	if anyOffline {
		return "alert"
	}
	return "ok"
}

// DeviceMapKey prefers a real IP; falls back to hostname when IP is the ingestion placeholder.
func DeviceMapKey(ipAddress, hostname string) string {
	ip := NormalizeIP(ipAddress)
	if ip != "" && ip != placeholderIP {
		return ip
	}
	if hostname != "" {
		return hostname
	}
	return ip
}

// NormalizeIP strips CIDR suffix from PostgreSQL inet text (e.g. "10.0.0.1/32").
func NormalizeIP(ipAddress string) string {
	if ipAddress == "" {
		return ""
	}
	// pgx may return CIDR-style values; strip prefix length if present.
	host, _, err := net.ParseCIDR(ipAddress)
	if err == nil {
		return host.String()
	}
	parsed := net.ParseIP(ipAddress)
	if parsed != nil {
		return parsed.String()
	}
	return ipAddress
}

// UptimeDays converts uptime_seconds to days. Returns nil when no sample.
func UptimeDays(uptimeSeconds *float64) *float64 {
	if uptimeSeconds == nil {
		return nil
	}
	days := *uptimeSeconds / 86400.0
	return &days
}

// LocationOrName returns location if set, otherwise the site name.
func LocationOrName(location, name string) string {
	if location != "" {
		return location
	}
	return name
}

// AvgNullable returns the average of non-nil values, or 0 when empty.
func AvgNullable(values []*float64) float64 {
	sum := 0.0
	n := 0
	for _, v := range values {
		if v == nil {
			continue
		}
		sum += *v
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}
