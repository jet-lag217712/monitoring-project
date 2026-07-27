package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const tokenBytes = 32

// ErrSessionNotFound indicates an absent, expired, or revoked session.
var ErrSessionNotFound = errors.New("session not found")

// Session is the server-side state associated with an opaque browser token.
type Session struct {
	Username  string
	CSRFHash  []byte
	ExpiresAt time.Time
}

// SessionStore persists only hashes of browser-visible secrets.
type SessionStore interface {
	Create(ctx context.Context, tokenHash []byte, username string, csrfHash []byte, expiresAt time.Time) error
	Find(ctx context.Context, tokenHash []byte, now time.Time) (Session, error)
	UpdateCSRF(ctx context.Context, tokenHash, csrfHash []byte) error
	Revoke(ctx context.Context, tokenHash []byte) error
	DeleteExpired(ctx context.Context, before time.Time) error
}

// PostgresSessionStore stores appliance sessions in PostgreSQL.
type PostgresSessionStore struct {
	pool *pgxpool.Pool
}

// NewPostgresSessionStore creates a PostgreSQL session store.
func NewPostgresSessionStore(pool *pgxpool.Pool) *PostgresSessionStore {
	return &PostgresSessionStore{pool: pool}
}

// Create inserts a new session.
func (s *PostgresSessionStore) Create(
	ctx context.Context,
	tokenHash []byte,
	username string,
	csrfHash []byte,
	expiresAt time.Time,
) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO appliance_sessions (token_hash, username, csrf_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`, tokenHash, username, csrfHash, expiresAt)
	if err != nil {
		return fmt.Errorf("create appliance session: %w", err)
	}
	return nil
}

// Find returns an active session.
func (s *PostgresSessionStore) Find(ctx context.Context, tokenHash []byte, now time.Time) (Session, error) {
	var session Session
	err := s.pool.QueryRow(ctx, `
		SELECT username, csrf_hash, expires_at
		FROM appliance_sessions
		WHERE token_hash = $1
			AND revoked_at IS NULL
			AND expires_at > $2
	`, tokenHash, now).Scan(&session.Username, &session.CSRFHash, &session.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("find appliance session: %w", err)
	}
	return session, nil
}

// UpdateCSRF rotates the CSRF secret associated with an active session.
func (s *PostgresSessionStore) UpdateCSRF(ctx context.Context, tokenHash, csrfHash []byte) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE appliance_sessions
		SET csrf_hash = $2
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash, csrfHash)
	if err != nil {
		return fmt.Errorf("update appliance session csrf: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrSessionNotFound
	}
	return nil
}

// Revoke invalidates a session immediately.
func (s *PostgresSessionStore) Revoke(ctx context.Context, tokenHash []byte) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE appliance_sessions
		SET revoked_at = COALESCE(revoked_at, NOW())
		WHERE token_hash = $1
	`, tokenHash)
	if err != nil {
		return fmt.Errorf("revoke appliance session: %w", err)
	}
	return nil
}

// DeleteExpired removes old session rows.
func (s *PostgresSessionStore) DeleteExpired(ctx context.Context, before time.Time) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM appliance_sessions
		WHERE expires_at <= $1 OR revoked_at <= $1
	`, before)
	if err != nil {
		return fmt.Errorf("delete expired appliance sessions: %w", err)
	}
	return nil
}

// SessionManager creates and validates appliance sessions.
type SessionManager struct {
	store  SessionStore
	broker Broker
	ttl    time.Duration
	random io.Reader
	now    func() time.Time
}

// NewSessionManager creates an appliance session manager.
func NewSessionManager(store SessionStore, broker Broker, ttl time.Duration) (*SessionManager, error) {
	if store == nil || broker == nil {
		return nil, errors.New("session store and broker are required")
	}
	if ttl <= 0 {
		return nil, errors.New("session ttl must be positive")
	}
	return &SessionManager{
		store:  store,
		broker: broker,
		ttl:    ttl,
		random: rand.Reader,
		now:    time.Now,
	}, nil
}

// Create authenticates credentials and creates a new opaque session.
func (m *SessionManager) Create(
	ctx context.Context,
	username string,
	password string,
) (string, string, Session, bool, error) {
	canonical, ok, err := m.broker.Authenticate(ctx, username, password)
	if err != nil || !ok {
		return "", "", Session{}, ok, err
	}

	token, err := randomToken(m.random)
	if err != nil {
		return "", "", Session{}, false, fmt.Errorf("generate session token: %w", err)
	}
	csrf, err := randomToken(m.random)
	if err != nil {
		return "", "", Session{}, false, fmt.Errorf("generate csrf token: %w", err)
	}
	session := Session{
		Username:  canonical,
		CSRFHash:  hashToken(csrf),
		ExpiresAt: m.now().Add(m.ttl),
	}
	if err := m.store.Create(ctx, hashToken(token), canonical, session.CSRFHash, session.ExpiresAt); err != nil {
		return "", "", Session{}, false, err
	}
	_ = m.store.DeleteExpired(ctx, m.now().Add(-24*time.Hour))
	return token, csrf, session, true, nil
}

// Validate verifies an opaque session and confirms the OS account is still enabled.
func (m *SessionManager) Validate(ctx context.Context, token string) (Session, error) {
	tokenHash, err := validatedTokenHash(token)
	if err != nil {
		return Session{}, ErrSessionNotFound
	}
	session, err := m.store.Find(ctx, tokenHash, m.now())
	if err != nil {
		return Session{}, err
	}
	active, err := m.broker.AccountStatus(ctx, session.Username)
	if err != nil {
		return Session{}, fmt.Errorf("check appliance account status: %w", err)
	}
	if !active {
		_ = m.store.Revoke(ctx, tokenHash)
		return Session{}, ErrSessionNotFound
	}
	return session, nil
}

// RotateCSRF replaces a session's CSRF secret.
func (m *SessionManager) RotateCSRF(ctx context.Context, token string) (string, error) {
	tokenHash, err := validatedTokenHash(token)
	if err != nil {
		return "", ErrSessionNotFound
	}
	csrf, err := randomToken(m.random)
	if err != nil {
		return "", fmt.Errorf("generate csrf token: %w", err)
	}
	if err := m.store.UpdateCSRF(ctx, tokenHash, hashToken(csrf)); err != nil {
		return "", err
	}
	return csrf, nil
}

// Revoke invalidates an opaque session.
func (m *SessionManager) Revoke(ctx context.Context, token string) error {
	tokenHash, err := validatedTokenHash(token)
	if err != nil {
		return nil
	}
	return m.store.Revoke(ctx, tokenHash)
}

func randomToken(source io.Reader) (string, error) {
	raw := make([]byte, tokenBytes)
	if _, err := io.ReadFull(source, raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func validatedTokenHash(token string) ([]byte, error) {
	if len(token) != base64.RawURLEncoding.EncodedLen(tokenBytes) {
		return nil, errors.New("invalid token length")
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != tokenBytes {
		return nil, errors.New("invalid token")
	}
	return hashToken(token), nil
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func csrfMatches(raw string, expectedHash []byte) bool {
	if _, err := validatedTokenHash(raw); err != nil || len(expectedHash) != sha256.Size {
		return false
	}
	actual := hashToken(raw)
	return subtle.ConstantTimeCompare(actual, expectedHash) == 1
}
