package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	SessionCookieName = "equate_session"
	CSRFHeaderName    = "X-CSRF-Token"
)

// ApplianceHandlers serves appliance-local authentication endpoints.
type ApplianceHandlers struct {
	sessions   *SessionManager
	rateLimit  *LoginRateLimiter
	log        *slog.Logger
	cookiePath string
	secure     bool
}

// ApplianceHandlersConfig configures cookie behavior for appliance auth routes.
type ApplianceHandlersConfig struct {
	Secure     bool
	CookiePath string
}

// NewApplianceHandlers creates appliance auth route handlers.
func NewApplianceHandlers(
	sessions *SessionManager,
	rateLimit *LoginRateLimiter,
	log *slog.Logger,
	cfg ApplianceHandlersConfig,
) *ApplianceHandlers {
	path := strings.TrimSpace(cfg.CookiePath)
	if path == "" {
		path = "/"
	}
	return &ApplianceHandlers{
		sessions:   sessions,
		rateLimit:  rateLimit,
		log:        log,
		cookiePath: path,
		secure:     cfg.Secure,
	}
}

// Register mounts POST /auth/login, POST /auth/logout, and GET /auth/me.
func (h *ApplianceHandlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/login", h.handleLogin)
	mux.HandleFunc("POST /auth/logout", h.handleLogout)
	mux.HandleFunc("GET /auth/me", h.handleMe)
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authUserResponse struct {
	Username  string `json:"username"`
	CSRFToken string `json:"csrf_token"`
}

func (h *ApplianceHandlers) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAuthError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	key := loginRateKey(clientIP(r), req.Username)
	if !h.rateLimit.Allow(key) {
		writeAuthError(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many login attempts")
		return
	}

	token, csrf, session, ok, err := h.sessions.Create(r.Context(), req.Username, req.Password)
	if err != nil {
		h.log.Info("login failed", "reason", "broker_error")
		writeAuthError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "authentication is temporarily unavailable")
		return
	}
	if !ok {
		h.rateLimit.RecordFailure(key)
		h.log.Info("login failed", "reason", "invalid_credentials")
		writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid username or password")
		return
	}

	h.rateLimit.Reset(key)
	h.setSessionCookie(w, token)
	writeJSONAuth(w, http.StatusOK, authUserResponse{
		Username:  session.Username,
		CSRFToken: csrf,
	})
}

func (h *ApplianceHandlers) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := sessionTokenFromRequest(r)
	if token == "" {
		writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	session, err := h.sessions.Validate(r.Context(), token)
	if err != nil {
		h.clearSessionCookie(w)
		writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}
	if !ValidateCSRFForSession(r, session) {
		writeAuthError(w, http.StatusForbidden, "FORBIDDEN", "invalid csrf token")
		return
	}

	_ = h.sessions.Revoke(r.Context(), token)
	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *ApplianceHandlers) handleMe(w http.ResponseWriter, r *http.Request) {
	token := sessionTokenFromRequest(r)
	if token == "" {
		writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	session, err := h.sessions.Validate(r.Context(), token)
	if err != nil {
		h.clearSessionCookie(w)
		writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	csrf, err := h.sessions.RotateCSRF(r.Context(), token)
	if err != nil {
		writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	writeJSONAuth(w, http.StatusOK, authUserResponse{
		Username:  session.Username,
		CSRFToken: csrf,
	})
}

func (h *ApplianceHandlers) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     h.cookiePath,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   0,
	})
}

func (h *ApplianceHandlers) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     h.cookiePath,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
	})
}

func sessionTokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie == nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

func loginRateKey(ip, username string) string {
	return strings.ToLower(strings.TrimSpace(ip)) + "|" + strings.ToLower(strings.TrimSpace(username))
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	host := strings.TrimSpace(r.RemoteAddr)
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		return host[:idx]
	}
	return host
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func writeJSONAuth(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	writeJSONAuth(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

// ValidateCSRFForSession checks the CSRF header against the active session.
func ValidateCSRFForSession(r *http.Request, session Session) bool {
	header := strings.TrimSpace(r.Header.Get(CSRFHeaderName))
	return csrfMatches(header, session.CSRFHash)
}

// ErrInvalidCSRF indicates a missing or invalid CSRF token.
var ErrInvalidCSRF = errors.New("invalid csrf token")
