package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

type staticGoogleVerifier struct{}

func (staticGoogleVerifier) Verify(context.Context, string) (*Claims, error) { return nil, nil }

func TestGoogleSessionAllowsOnlyVerifiedWorkspaceClaims(t *testing.T) {
	authenticator := &GoogleSessionAuthenticator{allowedDomains: normalizedSet([]string{"Example.COM"})}
	tests := []struct {
		name   string
		claims *Claims
		want   bool
	}{
		{name: "verified allowed workspace", claims: &Claims{Subject: "subject", Email: "operator@example.com", EmailVerified: true, HostedDomain: "example.com"}, want: true},
		{name: "unverified email", claims: &Claims{Subject: "subject", Email: "operator@example.com", HostedDomain: "example.com"}},
		{name: "consumer account has no hosted domain", claims: &Claims{Subject: "subject", Email: "operator@gmail.com", EmailVerified: true}},
		{name: "unapproved workspace", claims: &Claims{Subject: "subject", Email: "operator@other.example", EmailVerified: true, HostedDomain: "other.example"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := authenticator.allowed(test.claims); got != test.want {
				t.Fatalf("allowed() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestGoogleSessionUsesSecureHostCookie(t *testing.T) {
	authenticator := NewSessionAuthenticator(nil, SessionOptions{CookieName: "__Host-equate_session", Secure: true}, nil)
	cookie := authenticator.cookie(uuid.New(), time.Now().Add(time.Hour))
	if !cookie.HttpOnly || !cookie.Secure || cookie.Path != "/" || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("unexpected session cookie flags: %+v", cookie)
	}
}

func TestGoogleSessionRejectsNonHostCookie(t *testing.T) {
	_, err := NewGoogleSessionAuthenticator(nil, staticGoogleVerifier{}, GoogleSessionOptions{
		SessionOptions: SessionOptions{CookieName: "equate_session", Secure: true},
		ClientID:       "equate.apps.googleusercontent.com",
	}, nil)
	if err == nil {
		t.Fatal("NewGoogleSessionAuthenticator accepted a non-__Host- cookie")
	}
}

func TestBootstrapAuthenticatorLocksDashboardAccess(t *testing.T) {
	bootstrap := NewBootstrapAuthenticator()
	mux := http.NewServeMux()
	bootstrap.Register(mux)
	mux.HandleFunc("GET /api/sites", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := bootstrap.RequireSetup(mux)

	method := httptest.NewRecorder()
	handler.ServeHTTP(method, httptest.NewRequest(http.MethodGet, "/api/auth/method", nil))
	if method.Code != http.StatusOK {
		t.Fatalf("auth method status = %d, want 200", method.Code)
	}

	locked := httptest.NewRecorder()
	handler.ServeHTTP(locked, httptest.NewRequest(http.MethodGet, "/api/sites", nil))
	if locked.Code != http.StatusServiceUnavailable {
		t.Fatalf("dashboard status = %d, want 503", locked.Code)
	}
}
