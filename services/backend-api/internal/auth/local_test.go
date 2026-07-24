package auth

import (
	"net/http"
	"testing"
)

func TestArgon2idPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if !VerifyPassword(hash, "correct horse battery staple") {
		t.Fatal("expected correct password to verify")
	}
	if VerifyPassword(hash, "not the right password") {
		t.Fatal("incorrect password verified")
	}
}

func TestPublicSessionRoutesRemainMinimal(t *testing.T) {
	for _, route := range []struct {
		method string
		path   string
		public bool
	}{
		{http.MethodGet, "/api/auth/method", true},
		{http.MethodPost, "/api/auth/local/login", true},
		{http.MethodPost, "/api/auth/logout", true},
		{http.MethodGet, "/api/auth/oidc/start", true},
		{http.MethodGet, "/api/auth/oidc/callback", true},
		{http.MethodGet, "/api/auth/me", false},
		{http.MethodGet, "/api/sites", false},
	} {
		if got := isPublicSessionRoute(route.method, route.path); got != route.public {
			t.Errorf("isPublicSessionRoute(%s, %s) = %t, want %t", route.method, route.path, got, route.public)
		}
	}
}
