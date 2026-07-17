package control_test

import (
	"context"
	"os"
	"testing"

	"github.com/equate/ogsd/services/snmp-collector/internal/control"
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
