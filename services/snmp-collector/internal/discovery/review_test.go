package discovery

import (
	"testing"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
)

func TestAcceptReviewedSkipsAlreadyManagedDevices(t *testing.T) {
	current := []config.DeviceConfig{{
		ID:           "do-core",
		Host:         "10.255.0.1",
		Port:         161,
		CommunityEnv: "SNMP_COMMUNITY",
		Version:      "2c",
	}}
	reviews := []ReviewedCandidate{{
		Approved: true,
		Candidate: Candidate{
			IP:              "10.255.0.1",
			Result:          ProbeSucceeded,
			DetectedProfile: "cisco",
		},
		Device: config.DeviceConfig{
			ID:           "do-core",
			Host:         "10.255.0.1",
			Port:         161,
			CommunityEnv: "SNMP_COMMUNITY",
			Version:      "2c",
		},
	}}

	var writes int
	err := AcceptReviewed("/tmp/managed.yaml", current, nil, reviews, func(path string, devices []config.DeviceConfig) error {
		writes++
		t.Fatalf("unexpected write with %d devices", len(devices))
		return nil
	})
	if err != nil {
		t.Fatalf("AcceptReviewed: %v", err)
	}
	if writes != 0 {
		t.Fatalf("writes=%d", writes)
	}
}

func TestAcceptReviewedAppendsOnlyNewManagedDevices(t *testing.T) {
	current := []config.DeviceConfig{{
		ID:           "do-core",
		Host:         "10.255.0.1",
		Port:         161,
		CommunityEnv: "SNMP_COMMUNITY",
		Version:      "2c",
	}}
	reviews := []ReviewedCandidate{
		{
			Approved: true,
			Candidate: Candidate{
				IP:              "10.255.0.1",
				Result:          ProbeSucceeded,
				DetectedProfile: "cisco",
			},
			Device: config.DeviceConfig{
				ID:           "do-core",
				Host:         "10.255.0.1",
				Port:         161,
				CommunityEnv: "SNMP_COMMUNITY",
				Version:      "2c",
			},
		},
		{
			Approved: true,
			Candidate: Candidate{
				IP:              "10.255.0.11",
				Result:          ProbeSucceeded,
				DetectedProfile: "cisco",
			},
			Device: config.DeviceConfig{
				ID:           "site-a-mdf",
				Host:         "10.255.0.11",
				Port:         161,
				CommunityEnv: "SNMP_COMMUNITY",
				Version:      "2c",
			},
		},
	}

	var got []config.DeviceConfig
	err := AcceptReviewed("/tmp/managed.yaml", current, nil, reviews, func(path string, devices []config.DeviceConfig) error {
		got = append([]config.DeviceConfig(nil), devices...)
		return nil
	})
	if err != nil {
		t.Fatalf("AcceptReviewed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("device count=%d", len(got))
	}
	if got[0].ID != "do-core" || got[1].ID != "site-a-mdf" {
		t.Fatalf("devices=%v", got)
	}
}
