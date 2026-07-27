package setup

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func formatCandidateSummary(c map[string]any) string {
	ip := fmt.Sprint(c["ip"])
	host := strings.TrimSpace(fmt.Sprint(c["hostname"]))
	profile := fmt.Sprint(c["detected_profile"])
	result := fmt.Sprint(c["result"])
	parts := []string{ip, result}
	if host != "" {
		parts = append(parts, host)
	}
	if profile != "" {
		parts = append(parts, profile)
	}
	return strings.Join(parts, " · ")
}

func acceptApprovedCandidates(managedInventoryPath, communityEnv string, client controlCaller, candidates []map[string]any, approved []bool) (int, error) {
	reviews := make([]map[string]any, 0)
	for i, c := range candidates {
		if !approved[i] || fmt.Sprint(c["result"]) != "success" {
			continue
		}
		ip := fmt.Sprint(c["ip"])
		id := deviceIDFromDiscoveryCandidate(fmt.Sprint(c["hostname"]), ip)
		reviews = append(reviews, map[string]any{
			"approved": true,
			"candidate": map[string]any{
				"ip":               ip,
				"fingerprint":      c["fingerprint"],
				"detected_profile": c["detected_profile"],
				"hostname":         c["hostname"],
				"description":      c["description"],
				"result":           "success",
			},
			"device": map[string]any{
				"id":            id,
				"host":          ip,
				"port":          161,
				"community_env": communityEnv,
				"version":       "2c",
			},
		})
	}
	if len(reviews) == 0 {
		return 0, nil
	}
	if err := acceptReviews(managedInventoryPath, communityEnv, client, reviews); err != nil {
		return 0, err
	}
	return len(reviews), nil
}

func acceptReviews(managedInventoryPath, communityEnv string, client controlCaller, reviews []map[string]any) error {
	if len(reviews) == 0 {
		return fmt.Errorf("no candidates approved")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	prepare, err := client.Call(ctx, "ap1", "discovery.accept.prepare", map[string]any{"reviews": reviews})
	if err != nil {
		return err
	}
	if !prepare.OK {
		return fmt.Errorf("%s: %s", prepare.Error.Code, prepare.Error.Message)
	}
	commit, err := client.Call(ctx, "ap2", "discovery.accept.commit", map[string]any{
		"confirm_token": prepare.Result["confirm_token"],
		"revision":      prepare.Result["revision"],
	})
	if err != nil {
		return err
	}
	if !commit.OK {
		return fmt.Errorf("%s: %s", commit.Error.Code, commit.Error.Message)
	}
	if err := enrichManagedTopology(managedInventoryPath, communityEnv); err != nil {
		return err
	}
	reload, err := client.Call(ctx, "ap3", "config.reload", nil)
	if err != nil {
		return err
	}
	if !reload.OK {
		return fmt.Errorf("%s: %s", reload.Error.Code, reload.Error.Message)
	}
	return nil
}
