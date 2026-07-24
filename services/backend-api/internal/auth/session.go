package auth

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SessionOptions configures secure browser sessions for appliance-local users.
type SessionOptions struct {
	CookieName string
	TTL        time.Duration
	Secure     bool
}

// SessionAuthenticator owns local login and cookie validation.
type SessionAuthenticator struct {
	pool    *pgxpool.Pool
	options SessionOptions
	log     *slog.Logger
}

// NewSessionAuthenticator creates the local browser authentication service.
func NewSessionAuthenticator(pool *pgxpool.Pool, options SessionOptions, log *slog.Logger) *SessionAuthenticator {
	if options.CookieName == "" {
		options.CookieName = "__Host-equate_session"
	}
	if options.TTL <= 0 {
		options.TTL = 12 * time.Hour
	}
	if log == nil {
		log = slog.Default()
	}
	return &SessionAuthenticator{pool: pool, options: options, log: log}
}

// Register mounts the public session endpoints. Application configuration stays
// local-only; these endpoints never expose provider or secret configuration.
func (a *SessionAuthenticator) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/auth/method", a.handleMethod)
	mux.HandleFunc("POST /api/auth/local/login", a.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", a.handleLogout)
	mux.HandleFunc("GET /api/auth/me", a.handleMe)
}

func (a *SessionAuthenticator) handleMethod(w http.ResponseWriter, _ *http.Request) {
	writeAuthJSON(w, http.StatusOK, map[string]string{"provider": "local"})
}

func (a *SessionAuthenticator) handleLogin(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	if err := decoder.Decode(&input); err != nil {
		writeAuthError(w, http.StatusBadRequest, "invalid login request")
		return
	}
	user, err := AuthenticateLocal(r.Context(), a.pool, input.Username, input.Password)
	if err != nil {
		a.log.Info("local login rejected", "username", strings.TrimSpace(input.Username))
		writeAuthError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	sessionID, expires, err := CreateLocalSession(r.Context(), a.pool, user.ID, a.options.TTL)
	if err != nil {
		a.log.Error("create local session", "err", err)
		writeAuthError(w, http.StatusInternalServerError, "login unavailable")
		return
	}
	http.SetCookie(w, a.cookie(sessionID, expires))
	writeAuthJSON(w, http.StatusOK, map[string]any{"username": user.Username, "email": user.Email, "role": user.Role})
}

func (a *SessionAuthenticator) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(a.options.CookieName); err == nil {
		if id, err := uuid.Parse(cookie.Value); err == nil {
			if err := DeleteLocalSession(r.Context(), a.pool, id); err != nil {
				a.log.Warn("delete local session", "err", err)
			}
		}
	}
	cookie := a.cookie(uuid.Nil, time.Unix(0, 0))
	cookie.MaxAge = -1
	http.SetCookie(w, cookie)
	w.WriteHeader(http.StatusNoContent)
}

func (a *SessionAuthenticator) handleMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		writeAuthError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	writeAuthJSON(w, http.StatusOK, map[string]string{"subject": claims.Subject, "email": claims.Email, "name": claims.Name})
}

func (a *SessionAuthenticator) cookie(id uuid.UUID, expires time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     a.options.CookieName,
		Value:    id.String(),
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   a.options.Secure,
		SameSite: http.SameSiteStrictMode,
	}
}

// RequireSession protects all monitoring routes while allowing only the small
// public login surface through. Claims are available to future role checks.
func (a *SessionAuthenticator) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions || isPublicSessionRoute(r.Method, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie(a.options.CookieName)
		if err != nil {
			writeUnauthorized(w, "missing session")
			return
		}
		id, err := uuid.Parse(cookie.Value)
		if err != nil {
			writeUnauthorized(w, "invalid session")
			return
		}
		user, err := LocalSessionUser(r.Context(), a.pool, id)
		if err != nil {
			writeUnauthorized(w, "invalid session")
			return
		}
		claims := &Claims{Subject: user.ID.String(), Email: user.Email, Name: user.Username}
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// isPublicSessionRoute deliberately keeps the unauthenticated browser surface
// small. In particular, /api/auth/me is authenticated so it can be used to
// restore a browser session after a page reload.
func isPublicSessionRoute(method, path string) bool {
	return (method == http.MethodGet && path == "/api/auth/method") ||
		(method == http.MethodPost && path == "/api/auth/local/login") ||
		(method == http.MethodPost && path == "/api/auth/google/login") ||
		(method == http.MethodPost && path == "/api/auth/logout") ||
		(method == http.MethodGet && path == "/api/auth/oidc/start") ||
		(method == http.MethodGet && path == "/api/auth/oidc/callback")
}

func writeAuthJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	writeAuthJSON(w, status, map[string]any{"error": map[string]string{"code": "AUTH_ERROR", "message": message}})
}
