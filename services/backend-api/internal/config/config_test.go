package config

import "testing"

func TestDisabledAuthDoesNotProtectAPIRoutes(t *testing.T) {
	cfg := Config{Auth: AuthConfig{Enabled: false, Mode: "disabled"}}
	if cfg.AuthEnabled() {
		t.Fatal("disabled auth unexpectedly enabled route protection")
	}
}
