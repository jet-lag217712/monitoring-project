package auth

import "testing"

func TestOIDCAllowListRequiresExplicitEmailOrGroupMatch(t *testing.T) {
	authenticator := &OIDCSessionAuthenticator{
		emails: normalizedSet([]string{"Admin@Example.com"}),
		groups: normalizedSet([]string{"Equate-Operators"}),
	}
	if !authenticator.allowed("admin@example.com", nil) {
		t.Fatal("expected email allow-list to match")
	}
	if !authenticator.allowed("someone@example.com", []string{"equate-operators"}) {
		t.Fatal("expected group allow-list to match")
	}
	if authenticator.allowed("someone@example.com", []string{"guests"}) {
		t.Fatal("unexpected external identity allow")
	}
}
