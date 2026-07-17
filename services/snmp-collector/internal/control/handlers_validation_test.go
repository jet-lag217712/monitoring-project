package control_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"github.com/equate/ogsd/services/snmp-collector/internal/control"
	"github.com/equate/ogsd/services/snmp-collector/internal/health"
	"github.com/equate/ogsd/services/snmp-collector/internal/status"
)

func TestDependencyCommitRejectsMissingUpstream(t *testing.T) {
	env := startControlEnv(t)
	defer env.close()

	client := control.NewClient(env.socket)
	ctx := context.Background()

	prepare, err := client.Call(ctx, "1", "dependencies.prepare", map[string]any{
		"device_id":           "dev-001",
		"upstream_device_ids": []any{"missing-upstream"},
	})
	if err != nil || !prepare.OK {
		t.Fatalf("prepare: err=%v resp=%#v", err, prepare)
	}

	commit, err := client.Call(ctx, "2", "dependencies.commit", map[string]any{
		"confirm_token": prepare.Result["confirm_token"],
		"revision":      prepare.Result["revision"],
	})
	if err != nil {
		t.Fatal(err)
	}
	if commit.OK || commit.Error == nil || commit.Error.Code != control.CodeValidationFailed {
		t.Fatalf("expected validation failure, got %#v", commit)
	}
	if err := env.manager.Reload(); err != nil {
		t.Fatalf("reload should still succeed: %v", err)
	}
}

func TestThresholdCommitRejectsUnknownDevice(t *testing.T) {
	env := startControlEnv(t)
	defer env.close()

	client := control.NewClient(env.socket)
	ctx := context.Background()

	prepare, err := client.Call(ctx, "1", "thresholds.prepare", map[string]any{
		"device_id":             "phantom-device",
		"temperature_warning_c": 71.0,
	})
	if err != nil || !prepare.OK {
		t.Fatalf("prepare: err=%v resp=%#v", err, prepare)
	}

	commit, err := client.Call(ctx, "2", "thresholds.commit", map[string]any{
		"confirm_token": prepare.Result["confirm_token"],
		"revision":      prepare.Result["revision"],
	})
	if err != nil {
		t.Fatal(err)
	}
	if commit.OK || commit.Error == nil || commit.Error.Code != control.CodeValidationFailed {
		t.Fatalf("expected validation failure, got %#v", commit)
	}
}

func TestDependencyCommitRejectsCycleAcrossSequentialCommits(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "collector.yaml")
	managedPath := filepath.Join(root, "managed.yaml")
	socketPath := filepath.Join("/tmp", fmt.Sprintf("sc-cycle-%d.sock", time.Now().UnixNano()%1_000_000_000))
	auditPath := filepath.Join(root, "a.log")
	t.Cleanup(func() { _ = os.Remove(socketPath) })

	writeFile(t, configPath, "site_id: site-001\ncollector:\n  id: collector-001\ninventory:\n  managed_path: managed.yaml\nadmin:\n  listen: \"127.0.0.1:0\"\n  control_socket: "+strconvQuote(socketPath)+"\nhealth:\n  temperature_warning_c: 65\ndevices:\n  - id: dev-001\n    host: 127.0.0.1\n    community_env: SNMP_COMMUNITY_DEV_001\n  - id: dev-002\n    host: 127.0.0.2\n    community_env: SNMP_COMMUNITY_DEV_002\n")
	writeFile(t, managedPath, "devices: []\n")

	cfg, err := config.LoadForValidation(configPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	manager, err := config.NewManager(configPath, cfg)
	if err != nil {
		t.Fatal(err)
	}
	server, err := control.NewServer(control.Options{
		SocketPath: socketPath,
		Manager:    manager,
		Status:     status.New(),
		Health:     health.NewTracker(),
		Transport:  staticTransport{},
		AuditPath:  auditPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Listen(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = server.Serve(ctx) }()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	client := control.NewClient(socketPath)
	callCtx := context.Background()

	commitDependency := func(deviceID string, upstreams []any) control.Response {
		t.Helper()
		prepare, err := client.Call(callCtx, deviceID+"-prepare", "dependencies.prepare", map[string]any{
			"device_id":           deviceID,
			"upstream_device_ids": upstreams,
		})
		if err != nil || !prepare.OK {
			t.Fatalf("prepare %s: err=%v resp=%#v", deviceID, err, prepare)
		}
		commit, err := client.Call(callCtx, deviceID+"-commit", "dependencies.commit", map[string]any{
			"confirm_token": prepare.Result["confirm_token"],
			"revision":      prepare.Result["revision"],
		})
		if err != nil {
			t.Fatal(err)
		}
		return commit
	}

	first := commitDependency("dev-001", []any{"dev-002"})
	if !first.OK {
		t.Fatalf("first commit should succeed: %#v", first)
	}

	second := commitDependency("dev-002", []any{"dev-001"})
	if second.OK || second.Error == nil || second.Error.Code != control.CodeValidationFailed {
		t.Fatalf("expected cycle rejection, got %#v", second)
	}
	if err := manager.Reload(); err != nil {
		t.Fatalf("reload should still succeed: %v", err)
	}
}

func TestDependencyCommitDoesNotPersistInvalidState(t *testing.T) {
	env := startControlEnv(t)
	defer env.close()

	managedPath := env.manager.Current().ManagedInventoryPath()
	before, err := os.ReadFile(managedPath)
	if err != nil {
		t.Fatal(err)
	}

	client := control.NewClient(env.socket)
	ctx := context.Background()
	prepare, err := client.Call(ctx, "1", "dependencies.prepare", map[string]any{
		"device_id":           "dev-001",
		"upstream_device_ids": []any{"missing-upstream"},
	})
	if err != nil || !prepare.OK {
		t.Fatalf("prepare: err=%v resp=%#v", err, prepare)
	}
	_, _ = client.Call(ctx, "2", "dependencies.commit", map[string]any{
		"confirm_token": prepare.Result["confirm_token"],
		"revision":      prepare.Result["revision"],
	})

	after, err := os.ReadFile(managedPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("managed inventory changed after rejected commit:\nbefore=%q\nafter=%q", before, after)
	}
}
