package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// GoogleTokenVerifier validates the Google ID token returned by Google Identity
// Services. Keeping this boundary small makes the authorization policy testable
// without reaching Google's discovery endpoint.
type GoogleTokenVerifier interface {
	Verify(context.Context, string) (*Claims, error)
}

// GoogleSessionOptions describes the appliance browser-session policy.
// Google Workspace domain values are matched only against the verified hd claim.
type GoogleSessionOptions struct {
	SessionOptions
	ClientID       string
	AllowedDomains []string
}

// GoogleSessionAuthenticator exchanges a GIS ID token for a server-owned
// browser session. The browser never persists the Google credential.
type GoogleSessionAuthenticator struct {
	*SessionAuthenticator
	verifier       GoogleTokenVerifier
	clientID       string
	allowedDomains map[string]struct{}
}

// NewGoogleSessionAuthenticator constructs the Google GIS appliance flow.
func NewGoogleSessionAuthenticator(pool *pgxpool.Pool, verifier GoogleTokenVerifier, options GoogleSessionOptions, log *slog.Logger) (*GoogleSessionAuthenticator, error) {
	if verifier == nil {
		return nil, fmt.Errorf("google token verifier is required")
	}
	if strings.TrimSpace(options.ClientID) == "" {
		return nil, fmt.Errorf("google client id is required")
	}
	if options.CookieName == "" {
		options.CookieName = "__Host-equate_session"
	}
	if !strings.HasPrefix(options.CookieName, "__Host-") || !options.Secure {
		return nil, fmt.Errorf("google session requires a secure __Host- cookie")
	}
	return &GoogleSessionAuthenticator{
		SessionAuthenticator: NewSessionAuthenticator(pool, options.SessionOptions, log),
		verifier:             verifier,
		clientID:             strings.TrimSpace(options.ClientID),
		allowedDomains:       normalizedSet(options.AllowedDomains),
	}, nil
}

// Register mounts the minimal unauthenticated Google login surface.
func (a *GoogleSessionAuthenticator) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/auth/method", a.handleMethod)
	mux.HandleFunc("POST /api/auth/google/login", a.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", a.handleLogout)
	mux.HandleFunc("GET /api/auth/me", a.handleMe)
}

func (a *GoogleSessionAuthenticator) handleMethod(w http.ResponseWriter, _ *http.Request) {
	writeAuthJSON(w, http.StatusOK, map[string]string{
		"provider":  "google_session",
		"client_id": a.clientID,
	})
}

func (a *GoogleSessionAuthenticator) handleLogin(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Credential string `json:"credential"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	if err := decoder.Decode(&input); err != nil {
		writeAuthError(w, http.StatusBadRequest, "invalid google sign-in request")
		return
	}
	claims, err := a.verifier.Verify(r.Context(), input.Credential)
	if err != nil || !a.allowed(claims) {
		a.log.Info("google sign-in rejected", "hosted_domain", hostedDomain(claims))
		writeAuthError(w, http.StatusUnauthorized, "google identity is not allowed")
		return
	}
	user, err := FindOrCreateExternalUser(r.Context(), a.pool, "google", claims.Subject, claims.Email, claims.Name)
	if err != nil {
		a.log.Error("provision google identity", "err", err)
		writeAuthError(w, http.StatusInternalServerError, "sign-in unavailable")
		return
	}
	sessionID, expires, err := CreateLocalSession(r.Context(), a.pool, user.ID, a.options.TTL)
	if err != nil {
		a.log.Error("create google session", "err", err)
		writeAuthError(w, http.StatusInternalServerError, "sign-in unavailable")
		return
	}
	http.SetCookie(w, a.cookie(sessionID, expires))
	writeAuthJSON(w, http.StatusOK, map[string]string{"subject": user.ID.String(), "username": user.Username, "email": user.Email, "role": user.Role})
}

func (a *GoogleSessionAuthenticator) allowed(claims *Claims) bool {
	if claims == nil || strings.TrimSpace(claims.Subject) == "" || strings.TrimSpace(claims.Email) == "" || !claims.EmailVerified {
		return false
	}
	_, ok := a.allowedDomains[strings.ToLower(strings.TrimSpace(claims.HostedDomain))]
	return ok
}

func hostedDomain(claims *Claims) string {
	if claims == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(claims.HostedDomain))
}

// BootstrapAuthenticator keeps a new appliance closed until the local setup
// TUI has configured at least one approved Workspace domain.
type BootstrapAuthenticator struct{}

func NewBootstrapAuthenticator() *BootstrapAuthenticator { return &BootstrapAuthenticator{} }

func (a *BootstrapAuthenticator) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/auth/method", func(w http.ResponseWriter, _ *http.Request) {
		writeAuthJSON(w, http.StatusOK, map[string]string{"provider": "setup_required"})
	})
}

func (a *BootstrapAuthenticator) RequireSetup(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions || (r.Method == http.MethodGet && r.URL.Path == "/api/auth/method") {
			next.ServeHTTP(w, r)
			return
		}
		writeAuthError(w, http.StatusServiceUnavailable, "complete local appliance setup before dashboard access")
	})
}
