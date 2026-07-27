package auth_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/equate/ogsd/services/backend-api/internal/auth"
)

type memorySessionStore struct {
	sessions map[string]auth.Session
}

func newMemorySessionStore() *memorySessionStore {
	return &memorySessionStore{sessions: make(map[string]auth.Session)}
}

func (s *memorySessionStore) key(tokenHash []byte) string {
	return string(tokenHash)
}

func (s *memorySessionStore) Create(_ context.Context, tokenHash []byte, username string, csrfHash []byte, expiresAt time.Time) error {
	s.sessions[s.key(tokenHash)] = auth.Session{
		Username:  username,
		CSRFHash:  append([]byte(nil), csrfHash...),
		ExpiresAt: expiresAt,
	}
	return nil
}

func (s *memorySessionStore) Find(_ context.Context, tokenHash []byte, now time.Time) (auth.Session, error) {
	session, ok := s.sessions[s.key(tokenHash)]
	if !ok || !session.ExpiresAt.After(now) {
		return auth.Session{}, auth.ErrSessionNotFound
	}
	return session, nil
}

func (s *memorySessionStore) UpdateCSRF(_ context.Context, tokenHash, csrfHash []byte) error {
	session, ok := s.sessions[s.key(tokenHash)]
	if !ok {
		return auth.ErrSessionNotFound
	}
	session.CSRFHash = append([]byte(nil), csrfHash...)
	s.sessions[s.key(tokenHash)] = session
	return nil
}

func (s *memorySessionStore) Revoke(_ context.Context, tokenHash []byte) error {
	delete(s.sessions, s.key(tokenHash))
	return nil
}

func (s *memorySessionStore) DeleteExpired(context.Context, time.Time) error { return nil }

type stubBroker struct {
	authenticate func(username, password string) (string, bool, error)
	account      func(username string) (bool, error)
}

func (b stubBroker) Authenticate(_ context.Context, username, password string) (string, bool, error) {
	return b.authenticate(username, password)
}

func (b stubBroker) AccountStatus(_ context.Context, username string) (bool, error) {
	return b.account(username)
}

func TestSessionManagerCreateAndValidate(t *testing.T) {
	store := newMemorySessionStore()
	broker := stubBroker{
		authenticate: func(username, password string) (string, bool, error) {
			if username == "alice" && password == "secret" {
				return "alice", true, nil
			}
			return "", false, nil
		},
		account: func(username string) (bool, error) {
			return username == "alice", nil
		},
	}
	manager, err := auth.NewSessionManager(store, broker, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	token, csrf, session, ok, err := manager.Create(context.Background(), "alice", "secret")
	if err != nil || !ok || token == "" || csrf == "" || session.Username != "alice" {
		t.Fatalf("create ok=%v token=%q csrf=%q session=%+v err=%v", ok, token, csrf, session, err)
	}

	validated, err := manager.Validate(context.Background(), token)
	if err != nil || validated.Username != "alice" {
		t.Fatalf("validate username=%q err=%v", validated.Username, err)
	}
}

func TestSessionManagerRevokesDisabledAccount(t *testing.T) {
	store := newMemorySessionStore()
	active := true
	broker := stubBroker{
		authenticate: func(username, password string) (string, bool, error) {
			return username, password == "secret", nil
		},
		account: func(username string) (bool, error) {
			return active, nil
		},
	}
	manager, err := auth.NewSessionManager(store, broker, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	token, _, _, ok, err := manager.Create(context.Background(), "alice", "secret")
	if err != nil || !ok {
		t.Fatalf("create err=%v ok=%v", err, ok)
	}

	active = false
	if _, err := manager.Validate(context.Background(), token); err == nil {
		t.Fatal("expected disabled account to invalidate session")
	}
}

func TestApplianceLoginLogoutFlow(t *testing.T) {
	store := newMemorySessionStore()
	broker := stubBroker{
		authenticate: func(username, password string) (string, bool, error) {
			if username == "alice" && password == "secret" {
				return "alice", true, nil
			}
			return "", false, nil
		},
		account: func(username string) (bool, error) {
			return username == "alice", nil
		},
	}
	sessions, err := auth.NewSessionManager(store, broker, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	rateLimit := auth.NewLoginRateLimiter(3, time.Minute, 16)
	handlers := auth.NewApplianceHandlers(sessions, rateLimit, slog.New(slog.NewTextHandler(io.Discard, nil)), auth.ApplianceHandlersConfig{
		Secure: false,
	})

	mux := http.NewServeMux()
	handlers.Register(mux)

	loginBody, _ := json.Marshal(map[string]string{"username": "alice", "password": "secret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	mux.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", loginRec.Code, loginRec.Body.String())
	}

	cookie := loginRec.Result().Cookies()[0]
	var loginResp struct {
		Username  string `json:"username"`
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginResp); err != nil {
		t.Fatal(err)
	}

	meReq := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	meReq.AddCookie(cookie)
	meRec := httptest.NewRecorder()
	mux.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me status=%d body=%s", meRec.Code, meRec.Body.String())
	}
	var meResp struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(meRec.Body.Bytes(), &meResp); err != nil {
		t.Fatal(err)
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutReq.AddCookie(cookie)
	logoutReq.Header.Set(auth.CSRFHeaderName, meResp.CSRFToken)
	logoutRec := httptest.NewRecorder()
	mux.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("logout status=%d body=%s", logoutRec.Code, logoutRec.Body.String())
	}
}

func TestLoginRateLimiterBlocksAfterLimit(t *testing.T) {
	limiter := auth.NewLoginRateLimiter(2, time.Minute, 8)
	key := "127.0.0.1|alice"
	if !limiter.Allow(key) {
		t.Fatal("expected first attempt to be allowed")
	}
	limiter.RecordFailure(key)
	if !limiter.Allow(key) {
		t.Fatal("expected second attempt to be allowed")
	}
	limiter.RecordFailure(key)
	if limiter.Allow(key) {
		t.Fatal("expected third attempt to be blocked")
	}
	limiter.Reset(key)
	if !limiter.Allow(key) {
		t.Fatal("expected reset to clear failures")
	}
}

func TestLoginReturnsGenericFailure(t *testing.T) {
	store := newMemorySessionStore()
	broker := stubBroker{
		authenticate: func(_, _ string) (string, bool, error) {
			return "", false, nil
		},
		account: func(_ string) (bool, error) { return false, nil },
	}
	sessions, err := auth.NewSessionManager(store, broker, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	handlers := auth.NewApplianceHandlers(sessions, auth.NewLoginRateLimiter(5, time.Minute, 8), slog.New(slog.NewTextHandler(io.Discard, nil)), auth.ApplianceHandlersConfig{})
	mux := http.NewServeMux()
	handlers.Register(mux)

	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("wrong")) {
		t.Fatalf("response leaked credential detail: %s", rec.Body.Bytes())
	}
}

func TestBrokerClientRejectsOversizedPassword(t *testing.T) {
	client, err := auth.NewBrokerClient(filepathJoinTemp(t), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	password := make([]byte, 1100)
	if _, err := rand.Read(password); err != nil {
		t.Fatal(err)
	}
	_, _, err = client.Authenticate(context.Background(), "alice", string(password))
	if err == nil {
		t.Fatal("expected password validation error")
	}
}

func filepathJoinTemp(t *testing.T) string {
	t.Helper()
	return t.TempDir() + "/auth.sock"
}
