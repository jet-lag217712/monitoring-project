package auth_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/equate/ogsd/services/backend-api/internal/auth"
)

func TestRequireGoogleOIDCMissingBearer(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	h := auth.RequireGoogleOIDC(nil, slog.New(slog.NewTextHandler(io.Discard, nil)), next)

	req := httptest.NewRequest(http.MethodGet, "/api/sites", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRequireGoogleOIDCOptionsPassthrough(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	h := auth.RequireGoogleOIDC(nil, nil, next)

	req := httptest.NewRequest(http.MethodOptions, "/api/sites", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !called {
		t.Fatal("expected OPTIONS to reach next handler")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d", rec.Code)
	}
}
