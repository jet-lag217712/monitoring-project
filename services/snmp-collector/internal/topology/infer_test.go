package topology

import (
	"context"
	"testing"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
)

func TestEnrichNamingFallback(t *testing.T) {
	devices := []config.DeviceConfig{
		{ID: "do-core", Host: "10.255.0.1"},
		{ID: "site-a-mdf", Host: "10.255.0.11"},
		{ID: "site-a-idf1", Host: "10.255.0.12"},
	}
	out := Enrich(devices, "public", func(context.Context, string, string) ([]string, error) {
		return nil, context.DeadlineExceeded
	})
	if out[0].Role != "Core Switch" || len(out[0].UpstreamDeviceIDs) != 0 {
		t.Fatalf("core: role=%q upstream=%v", out[0].Role, out[0].UpstreamDeviceIDs)
	}
	if out[1].Role != "Distribution Switch" || len(out[1].UpstreamDeviceIDs) != 1 || out[1].UpstreamDeviceIDs[0] != "do-core" {
		t.Fatalf("mdf: role=%q upstream=%v", out[1].Role, out[1].UpstreamDeviceIDs)
	}
	if out[2].Role != "Access Switch" || len(out[2].UpstreamDeviceIDs) != 1 || out[2].UpstreamDeviceIDs[0] != "site-a-mdf" {
		t.Fatalf("idf: role=%q upstream=%v", out[2].Role, out[2].UpstreamDeviceIDs)
	}
}

func TestEnrichCDPBuildsTree(t *testing.T) {
	devices := []config.DeviceConfig{
		{ID: "do-core", Host: "10.255.0.1"},
		{ID: "site-a-mdf", Host: "10.255.0.11"},
		{ID: "site-a-idf1", Host: "10.255.0.12"},
	}
	links := map[string][]string{
		"10.255.0.1":  {"10.255.0.11"},
		"10.255.0.11": {"10.255.0.1", "10.255.0.12"},
		"10.255.0.12": {"10.255.0.11"},
	}
	out := Enrich(devices, "public", func(_ context.Context, host, _ string) ([]string, error) {
		return links[host], nil
	})
	if out[2].UpstreamDeviceIDs[0] != "site-a-mdf" {
		t.Fatalf("idf upstream=%v, want site-a-mdf", out[2].UpstreamDeviceIDs)
	}
}
