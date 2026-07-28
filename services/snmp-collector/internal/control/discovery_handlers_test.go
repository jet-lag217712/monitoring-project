package control_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/control"
)

func TestDiscoveryPolicyPrepareCommitReload(t *testing.T) {
	env := startControlEnv(t)
	defer env.close()

	client := control.NewClient(env.socket)
	ctx := context.Background()

	prepare, err := client.Call(ctx, "1", "discovery.policy.prepare", map[string]any{
		"allowed_cidrs":         []any{"10.0.0.0/24"},
		"community_env":           "SNMP_DISCOVERY_COMMUNITY",
		"max_probes_per_second": 6.0,
		"probe_burst":             2.0,
	})
	if err != nil || !prepare.OK {
		t.Fatalf("prepare: err=%v resp=%#v", err, prepare)
	}

	commit, err := client.Call(ctx, "2", "discovery.policy.commit", map[string]any{
		"confirm_token": prepare.Result["confirm_token"],
		"revision":      prepare.Result["revision"],
	})
	if err != nil || !commit.OK {
		t.Fatalf("commit: err=%v resp=%#v", err, commit)
	}

	reload, err := client.Call(ctx, "3", "config.reload", nil)
	if err != nil || !reload.OK {
		t.Fatalf("reload: err=%v resp=%#v", err, reload)
	}

	cfg := env.manager.Current()
	if len(cfg.Discovery.AllowedCIDRs) != 1 || cfg.Discovery.AllowedCIDRs[0] != "10.0.0.0/24" {
		t.Fatalf("cidrs=%#v", cfg.Discovery.AllowedCIDRs)
	}
	if cfg.Discovery.MaxProbesPerSecond != 6 || cfg.Discovery.ProbeBurst != 2 {
		t.Fatalf("rate=%v burst=%v", cfg.Discovery.MaxProbesPerSecond, cfg.Discovery.ProbeBurst)
	}

	auditData, err := os.ReadFile(env.audit)
	if err != nil {
		t.Fatal(err)
	}
	auditText := string(auditData)
	if !strings.Contains(auditText, `"action":"discovery.policy.commit"`) {
		t.Fatalf("audit missing policy commit: %s", auditText)
	}
	if strings.Contains(strings.ToLower(auditText), "password") || strings.Contains(auditText, "public") {
		t.Fatalf("audit must not contain secrets: %s", auditText)
	}
}

func TestDiscoveryScanRejectsMissingCIDRs(t *testing.T) {
	env := startControlEnv(t)
	defer env.close()

	client := control.NewClient(env.socket)
	ctx := context.Background()

	resp, err := client.Call(ctx, "1", "discovery.scan.start", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.OK || resp.Error == nil || resp.Error.Code != control.CodeValidationFailed {
		t.Fatalf("expected validation failure, got %#v", resp)
	}
}

func TestDiscoveryScanAsyncAndProgress(t *testing.T) {
	env := startControlEnvWithDiscovery(t)
	defer env.close()

	client := control.NewClient(env.socket)
	ctx := context.Background()

	t.Setenv("SNMP_DISCOVERY_COMMUNITY", "public")

	start, err := client.Call(ctx, "1", "discovery.scan.start", map[string]any{"async": true})
	if err != nil || !start.OK {
		t.Fatalf("async start: err=%v resp=%#v", err, start)
	}
	if start.Result["running"] != true {
		t.Fatalf("running=%v", start.Result["running"])
	}
	if got, _ := start.Result["target_count"].(float64); got != 4 {
		t.Fatalf("target_count=%v", start.Result["target_count"])
	}

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		progress, err := client.Call(ctx, "2", "discovery.scan.progress", nil)
		if err != nil || !progress.OK {
			t.Fatalf("progress: err=%v resp=%#v", err, progress)
		}
		if progress.Result["running"] != true {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	final, err := client.Call(ctx, "3", "discovery.scan.progress", nil)
	if err != nil || !final.OK {
		t.Fatalf("final progress: err=%v resp=%#v", err, final)
	}
	if final.Result["running"] == true {
		t.Fatal("scan still running after deadline")
	}
	if got, _ := final.Result["probed"].(float64); got != 4 {
		t.Fatalf("probed=%v", final.Result["probed"])
	}
}

func TestDiscoveryScanRejectsConcurrentAsync(t *testing.T) {
	env := startControlEnvWithDiscovery(t)
	defer env.close()

	client := control.NewClient(env.socket)
	ctx := context.Background()
	t.Setenv("SNMP_DISCOVERY_COMMUNITY", "public")

	first, err := client.Call(ctx, "1", "discovery.scan.start", map[string]any{"async": true})
	if err != nil || !first.OK {
		t.Fatalf("first start: err=%v resp=%#v", err, first)
	}

	second, err := client.Call(ctx, "2", "discovery.scan.start", map[string]any{"async": true})
	if err != nil {
		t.Fatal(err)
	}
	if second.OK || second.Error == nil || second.Error.Code != control.CodeConflict {
		t.Fatalf("expected conflict, got %#v", second)
	}

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		progress, err := client.Call(ctx, "3", "discovery.scan.progress", nil)
		if err != nil || !progress.OK {
			t.Fatalf("progress: err=%v resp=%#v", err, progress)
		}
		if progress.Result["running"] != true {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("first scan did not finish")
}

func startControlEnvWithDiscovery(t *testing.T) *controlEnv {
	env := startControlEnv(t)
	client := control.NewClient(env.socket)
	ctx := context.Background()

	prepare, err := client.Call(ctx, "1", "discovery.policy.prepare", map[string]any{
		"allowed_cidrs":         []any{"192.0.2.0/30"},
		"community_env":         "SNMP_DISCOVERY_COMMUNITY",
		"max_probes_per_second": 1000.0,
		"probe_burst":           4.0,
	})
	if err != nil || !prepare.OK {
		t.Fatalf("prepare: err=%v resp=%#v", err, prepare)
	}
	commit, err := client.Call(ctx, "2", "discovery.policy.commit", map[string]any{
		"confirm_token": prepare.Result["confirm_token"],
		"revision":      prepare.Result["revision"],
	})
	if err != nil || !commit.OK {
		t.Fatalf("commit: err=%v resp=%#v", err, commit)
	}
	reload, err := client.Call(ctx, "3", "config.reload", nil)
	if err != nil || !reload.OK {
		t.Fatalf("reload: err=%v resp=%#v", err, reload)
	}
	return env
}

func TestDiscoveryAcceptPrepareCommit(t *testing.T) {
	env := startControlEnv(t)
	defer env.close()

	client := control.NewClient(env.socket)
	ctx := context.Background()

	reviews := []any{
		map[string]any{
			"approved": true,
			"candidate": map[string]any{
				"ip":               "10.0.0.50",
				"fingerprint":      "fp-1",
				"detected_profile": "core",
				"result":           "success",
			},
			"device": map[string]any{
				"id":            "discovered-001",
				"host":          "10.0.0.50",
				"port":          161,
				"community_env": "SNMP_COMMUNITY_DEV_001",
				"version":       "2c",
			},
		},
	}

	prepare, err := client.Call(ctx, "1", "discovery.accept.prepare", map[string]any{
		"reviews": reviews,
	})
	if err != nil || !prepare.OK {
		t.Fatalf("prepare: err=%v resp=%#v", err, prepare)
	}

	commit, err := client.Call(ctx, "2", "discovery.accept.commit", map[string]any{
		"confirm_token": prepare.Result["confirm_token"],
		"revision":      prepare.Result["revision"],
	})
	if err != nil || !commit.OK {
		t.Fatalf("commit: err=%v resp=%#v", err, commit)
	}

	reload, err := client.Call(ctx, "3", "config.reload", nil)
	if err != nil || !reload.OK {
		t.Fatalf("reload: err=%v resp=%#v", err, reload)
	}

	found := false
	for _, d := range env.manager.Current().Devices {
		if d.ID == "discovered-001" {
			found = true
			if d.Host != "10.0.0.50" {
				t.Fatalf("host=%q", d.Host)
			}
		}
	}
	if !found {
		t.Fatal("discovered device not in active inventory after reload")
	}
}
