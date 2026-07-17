package validate_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/equate/ogsd/services/ingestion-service/internal/validate"
)

func schemaExample(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// services/ingestion-service/internal/validate -> repo root
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
	path := filepath.Join(root, "docs", "schemas", "snmp-collector-v2", "examples", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read example %s: %v", name, err)
	}
	return data
}

func TestValidateV2_SchemaExamples(t *testing.T) {
	cases := []struct {
		name  string
		topic string
		file  string
		kind  validate.Kind
	}{
		{
			name:  "device",
			topic: "site/site-001/device/dist-01/telemetry/v2/device",
			file:  "device-event.json",
			kind:  validate.KindDeviceV2,
		},
		{
			name:  "interface",
			topic: "site/site-001/device/dist-01/telemetry/v2/interface",
			file:  "interface-event.json",
			kind:  validate.KindInterfaceV2,
		},
		{
			name:  "health",
			topic: "site/site-001/device/access-01/telemetry/v2/health",
			file:  "health-event.json",
			kind:  validate.KindHealthV2,
		},
		{
			name:  "heartbeat",
			topic: "site/site-001/collector/collector-west-01/telemetry/v2/heartbeat",
			file:  "heartbeat-event.json",
			kind:  validate.KindHeartbeatV2,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg, err := validate.Validate(tc.topic, schemaExample(t, tc.file))
			if err != nil {
				t.Fatalf("validate: %v", err)
			}
			if msg.Kind != tc.kind {
				t.Fatalf("kind=%s want %s", msg.Kind, tc.kind)
			}
		})
	}
}

func TestValidateV2_RejectsSchemaVersion(t *testing.T) {
	payload := []byte(`{
		"schema_version":"1.0",
		"event_id":"018f3e2c-7a9d-7b20-8f63-1e2d3c4b5a63",
		"event_type":"collector_heartbeat",
		"site_id":"site-001",
		"collector_id":"collector-west-01",
		"observed_at":"2026-07-16T18:00:00Z",
		"emitted_at":"2026-07-16T18:00:01Z",
		"config_revision":"rev-1",
		"payload":{"hostname":"h","version":"v","git_commit":"c","build_time":"unknown","uptime_seconds":1,"sqlite_queue_depth":0,"memory_usage_bytes":1,"goroutine_count":1}
	}`)
	_, err := validate.Validate("site/site-001/collector/collector-west-01/telemetry/v2/heartbeat", payload)
	if err == nil {
		t.Fatal("expected schema_version rejection")
	}
}

func TestValidateV2_RejectsTopicBodyMismatch(t *testing.T) {
	payload := schemaExample(t, "device-event.json")
	_, err := validate.Validate("site/site-001/device/other-device/telemetry/v2/device", payload)
	if err == nil {
		t.Fatal("expected topic/body device_id mismatch")
	}
}

func TestValidateV2_RejectsInvalidHealthState(t *testing.T) {
	payload := []byte(`{
		"schema_version":"2.0",
		"event_id":"018f3e2c-7a9d-7b20-8f63-1e2d3c4b5a62",
		"event_type":"health_state",
		"site_id":"site-001",
		"collector_id":"collector-west-01",
		"device_id":"access-01",
		"observed_at":"2026-07-16T18:00:00Z",
		"emitted_at":"2026-07-16T18:00:02Z",
		"config_revision":"rev-1",
		"payload":{
			"state":"degraded",
			"reason":"reachable",
			"transition":"initial",
			"failure_count":0,
			"failure_threshold":2,
			"upstream_device_ids":[],
			"unavailable_upstream_device_ids":[],
			"root_cause_device_ids":[]
		}
	}`)
	_, err := validate.Validate("site/site-001/device/access-01/telemetry/v2/health", payload)
	if err == nil {
		t.Fatal("expected invalid health state rejection")
	}
}

func TestValidateV2_RejectsExtraEnvelopeField(t *testing.T) {
	payload := []byte(`{
		"schema_version":"2.0",
		"event_id":"018f3e2c-7a9d-7b20-8f63-1e2d3c4b5a63",
		"event_type":"collector_heartbeat",
		"site_id":"site-001",
		"collector_id":"collector-west-01",
		"observed_at":"2026-07-16T18:00:00Z",
		"emitted_at":"2026-07-16T18:00:01Z",
		"config_revision":"rev-1",
		"secret":"nope",
		"payload":{"hostname":"h","version":"v","git_commit":"c","build_time":"unknown","uptime_seconds":1,"sqlite_queue_depth":0,"memory_usage_bytes":1,"goroutine_count":1}
	}`)
	_, err := validate.Validate("site/site-001/collector/collector-west-01/telemetry/v2/heartbeat", payload)
	if err == nil {
		t.Fatal("expected extra field rejection")
	}
}
