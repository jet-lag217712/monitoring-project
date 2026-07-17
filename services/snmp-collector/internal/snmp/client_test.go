package snmp

import (
	"strings"
	"testing"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
)

func TestNewClientResolvesCommunityAtRuntime(t *testing.T) {
	t.Setenv("SNMP_TEST_COMMUNITY", "secret-community")
	client, err := NewClient(config.DeviceConfig{
		Host:         "127.0.0.1",
		Port:         161,
		Version:      "2c",
		CommunityEnv: "SNMP_TEST_COMMUNITY",
	}, config.SNMPConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client.params.Community != "secret-community" {
		t.Fatalf("community=%q", client.params.Community)
	}
}

func TestNewClientDoesNotExposeMissingCommunityValue(t *testing.T) {
	t.Setenv("SNMP_TEST_COMMUNITY", "")
	_, err := NewClient(config.DeviceConfig{
		Host:         "127.0.0.1",
		Port:         161,
		Version:      "2c",
		CommunityEnv: "SNMP_TEST_COMMUNITY",
	}, config.SNMPConfig{})
	if err == nil || !strings.Contains(err.Error(), "SNMP_TEST_COMMUNITY") {
		t.Fatalf("error=%v", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("error leaked a secret: %v", err)
	}
}
