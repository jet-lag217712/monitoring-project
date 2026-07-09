package derive

import (
	"net"
	"time"
)

const placeholderIP = "0.0.0.0"

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

// DeviceStatusCode returns frontend numeric status: 1 healthy, 3 critical.
func DeviceStatusCode(online bool) int {
	if online {
		return 1
	}
	return 3
}

// SiteStatus returns "ok" or "alert" from device online flags.
// No "caution" until utilization metrics exist.
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
