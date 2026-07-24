package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
)

const oidcStateCookie = "__Host-equate_oidc_state"

// OIDCOptions is the non-secret browser authentication configuration. The
// client secret is rendered to the process environment before this is built.
type OIDCOptions struct {
	Provider      string
	Issuer        string
	ClientID      string
	ClientSecret  string
	RedirectURL   string
	AllowedEmails []string
	AllowedGroups []string
	CookieName    string
	SessionTTL    time.Duration
	CookieSecure  bool
}

// OIDCSessionAuthenticator implements generic OIDC and Google browser login.
// LDAP/AD can be placed behind a standards-compliant OIDC provider while their
// credentials remain in the appliance secret store.
type OIDCSessionAuthenticator struct {
	*SessionAuthenticator
	provider string
	oauth    oauth2.Config
	verifier *oidc.IDTokenVerifier
	emails   map[string]struct{}
	groups   map[string]struct{}
	log      *slog.Logger
}

func NewOIDCSessionAuthenticator(ctx context.Context, pool *pgxpool.Pool, options OIDCOptions, log *slog.Logger) (*OIDCSessionAuthenticator, error) {
	if strings.TrimSpace(options.Provider) == "" || strings.TrimSpace(options.Issuer) == "" || strings.TrimSpace(options.ClientID) == "" || strings.TrimSpace(options.ClientSecret) == "" || strings.TrimSpace(options.RedirectURL) == "" {
		return nil, fmt.Errorf("OIDC provider, issuer, client ID, client secret, and redirect URL are required")
	}
	if len(options.AllowedEmails) == 0 && len(options.AllowedGroups) == 0 {
		return nil, fmt.Errorf("OIDC allow-list is required")
	}
	if parsed, err := url.ParseRequestURI(options.RedirectURL); err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil, fmt.Errorf("OIDC redirect URL must be an absolute HTTPS URL")
	}
	if log == nil {
		log = slog.Default()
	}
	provider, err := oidc.NewProvider(ctx, options.Issuer)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC provider: %w", err)
	}
	a := &OIDCSessionAuthenticator{
		SessionAuthenticator: NewSessionAuthenticator(pool, SessionOptions{CookieName: options.CookieName, TTL: options.SessionTTL, Secure: options.CookieSecure}, log),
		provider:             options.Provider,
		oauth: oauth2.Config{
			ClientID: options.ClientID, ClientSecret: options.ClientSecret, RedirectURL: options.RedirectURL,
			Endpoint: provider.Endpoint(), Scopes: []string{oidc.ScopeOpenID, "profile", "email", "groups"},
		},
		verifier: provider.Verifier(&oidc.Config{ClientID: options.ClientID}),
		emails:   normalizedSet(options.AllowedEmails),
		groups:   normalizedSet(options.AllowedGroups),
		log:      log,
	}
	return a, nil
}

func (a *OIDCSessionAuthenticator) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/auth/method", a.handleMethod)
	mux.HandleFunc("GET /api/auth/oidc/start", a.handleStart)
	mux.HandleFunc("GET /api/auth/oidc/callback", a.handleCallback)
	mux.HandleFunc("POST /api/auth/logout", a.handleLogout)
	mux.HandleFunc("GET /api/auth/me", a.handleMe)
}

func (a *OIDCSessionAuthenticator) handleMethod(w http.ResponseWriter, _ *http.Request) {
	writeAuthJSON(w, http.StatusOK, map[string]string{"provider": "oidc"})
}

func (a *OIDCSessionAuthenticator) handleStart(w http.ResponseWriter, r *http.Request) {
	state, err := randomOIDCValue()
	if err != nil {
		writeAuthError(w, http.StatusInternalServerError, "sign-in unavailable")
		return
	}
	nonce, err := randomOIDCValue()
	if err != nil {
		writeAuthError(w, http.StatusInternalServerError, "sign-in unavailable")
		return
	}
	verifier, err := randomOIDCValue()
	if err != nil {
		writeAuthError(w, http.StatusInternalServerError, "sign-in unavailable")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: oidcStateCookie, Value: state + "." + nonce + "." + verifier, Path: "/", HttpOnly: true, Secure: a.options.Secure, SameSite: http.SameSiteLaxMode, MaxAge: 600})
	http.Redirect(w, r, a.oauth.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)), http.StatusFound)
}

func (a *OIDCSessionAuthenticator) handleCallback(w http.ResponseWriter, r *http.Request) {
	if errCode := r.URL.Query().Get("error"); errCode != "" {
		writeAuthError(w, http.StatusUnauthorized, "identity provider rejected sign-in")
		return
	}
	state, nonce, verifier, ok := a.callbackState(r)
	if !ok || r.URL.Query().Get("state") != state || r.URL.Query().Get("code") == "" {
		writeAuthError(w, http.StatusUnauthorized, "invalid sign-in response")
		return
	}
	token, err := a.oauth.Exchange(r.Context(), r.URL.Query().Get("code"), oauth2.VerifierOption(verifier))
	if err != nil {
		a.log.Info("OIDC code exchange rejected", "provider", a.provider, "err", err)
		writeAuthError(w, http.StatusUnauthorized, "identity provider sign-in failed")
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		writeAuthError(w, http.StatusUnauthorized, "identity provider did not return an ID token")
		return
	}
	idToken, err := a.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		writeAuthError(w, http.StatusUnauthorized, "invalid identity provider token")
		return
	}
	var claims struct {
		Subject       string   `json:"sub"`
		Email         string   `json:"email"`
		EmailVerified bool     `json:"email_verified"`
		Name          string   `json:"name"`
		Groups        []string `json:"groups"`
		Nonce         string   `json:"nonce"`
	}
	if err := idToken.Claims(&claims); err != nil || claims.Subject == "" || claims.Nonce != nonce || !claims.EmailVerified || !a.allowed(claims.Email, claims.Groups) {
		writeAuthError(w, http.StatusUnauthorized, "identity is not allowed")
		return
	}
	user, err := FindOrCreateExternalUser(r.Context(), a.pool, a.provider, claims.Subject, claims.Email, claims.Name)
	if err != nil {
		a.log.Error("provision external identity", "provider", a.provider, "err", err)
		writeAuthError(w, http.StatusInternalServerError, "sign-in unavailable")
		return
	}
	sessionID, expires, err := CreateLocalSession(r.Context(), a.pool, user.ID, a.options.TTL)
	if err != nil {
		writeAuthError(w, http.StatusInternalServerError, "sign-in unavailable")
		return
	}
	http.SetCookie(w, a.cookie(sessionID, expires))
	http.SetCookie(w, &http.Cookie{Name: oidcStateCookie, Value: "", Path: "/", HttpOnly: true, Secure: a.options.Secure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (a *OIDCSessionAuthenticator) callbackState(r *http.Request) (string, string, string, bool) {
	cookie, err := r.Cookie(oidcStateCookie)
	if err != nil {
		return "", "", "", false
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 3 || len(parts[0]) < 32 || len(parts[1]) < 32 || len(parts[2]) < 32 {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

func (a *OIDCSessionAuthenticator) allowed(email string, groups []string) bool {
	if _, ok := a.emails[strings.ToLower(strings.TrimSpace(email))]; ok {
		return true
	}
	for _, group := range groups {
		if _, ok := a.groups[strings.ToLower(strings.TrimSpace(group))]; ok {
			return true
		}
	}
	return false
}

func normalizedSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value = strings.ToLower(strings.TrimSpace(value)); value != "" {
			set[value] = struct{}{}
		}
	}
	return set
}

func randomOIDCValue() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}
