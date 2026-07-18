package control_test

import (
	"context"
	"os"
	"strings"
	"testing"

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
