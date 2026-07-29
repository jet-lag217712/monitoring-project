package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDisabledAuthenticatorReportsNoAuthAndLeavesAPIRoutesOpen(t *testing.T) {
	mux := http.NewServeMux()
	NewDisabledAuthenticator().Register(mux)
	mux.HandleFunc("GET /api/sites", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	method := httptest.NewRecorder()
	mux.ServeHTTP(method, httptest.NewRequest(http.MethodGet, "/api/auth/method", nil))
	if method.Code != http.StatusOK || method.Body.String() != "{\"provider\":\"disabled\"}\n" {
		t.Fatalf("auth method response = status %d body %q", method.Code, method.Body.String())
	}

	sites := httptest.NewRecorder()
	mux.ServeHTTP(sites, httptest.NewRequest(http.MethodGet, "/api/sites", nil))
	if sites.Code != http.StatusOK {
		t.Fatalf("unauthenticated API route status = %d, want %d", sites.Code, http.StatusOK)
	}
}
