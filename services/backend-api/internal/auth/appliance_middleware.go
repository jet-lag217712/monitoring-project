package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
)

type applianceUserKey struct{}

// ApplianceUser is the authenticated appliance-local identity.
type ApplianceUser struct {
	Username string
}

// ApplianceUserFromContext returns the authenticated appliance user when present.
func ApplianceUserFromContext(ctx context.Context) (ApplianceUser, bool) {
	user, ok := ctx.Value(applianceUserKey{}).(ApplianceUser)
	return user, ok
}

// RequireApplianceSession protects /api routes with an opaque session cookie.
func RequireApplianceSession(sessions *SessionManager, log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		token := sessionTokenFromRequest(r)
		if token == "" {
			writeUnauthorized(w, "not authenticated")
			return
		}

		session, err := sessions.Validate(r.Context(), token)
		if err != nil {
			if log != nil {
				log.Info("session rejected")
			}
			writeUnauthorized(w, "not authenticated")
			return
		}

		ctx := context.WithValue(r.Context(), applianceUserKey{}, ApplianceUser{Username: session.Username})
		ctx = context.WithValue(ctx, sessionContextKey{}, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type sessionContextKey struct{}

// SessionFromContext returns the validated appliance session when present.
func SessionFromContext(ctx context.Context) (Session, bool) {
	session, ok := ctx.Value(sessionContextKey{}).(Session)
	return session, ok
}

// RequireApplianceCSRF enforces double-submit CSRF protection for mutating requests.
func RequireApplianceCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions || isSafeMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/auth/login") {
			next.ServeHTTP(w, r)
			return
		}

		session, ok := SessionFromContext(r.Context())
		if !ok {
			writeUnauthorized(w, "not authenticated")
			return
		}
		if !ValidateCSRFForSession(r, session) {
			writeAuthError(w, http.StatusForbidden, "FORBIDDEN", "invalid csrf token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead:
		return true
	default:
		return false
	}
}
