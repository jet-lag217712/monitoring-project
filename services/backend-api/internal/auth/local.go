package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/argon2"
)

const (
	argonMemory  = 64 * 1024
	argonTime    = 3
	argonThreads = 2
	argonKeyLen  = 32
)

// LocalUser is the only v1 application role principal.
type LocalUser struct {
	ID       uuid.UUID
	Username string
	Email    string
	Role     string
}

// HashPassword returns a self-describing Argon2id encoded password hash.
func HashPassword(password string) (string, error) {
	if len(password) < 12 {
		return "", fmt.Errorf("password must be at least 12 characters")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash)), nil
}

// VerifyPassword validates a password against the encoded Argon2id hash.
func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}
	var memory uint32
	var iterations uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &threads); err != nil || memory == 0 || iterations == 0 || threads == 0 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 16 {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(want) == 0 {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, iterations, memory, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

// AuthenticateLocal verifies a local user without leaking whether a username
// exists. It returns an authentication failure for disabled users as well.
func AuthenticateLocal(ctx context.Context, pool *pgxpool.Pool, username, password string) (*LocalUser, error) {
	if pool == nil {
		return nil, fmt.Errorf("database pool is required")
	}
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, fmt.Errorf("invalid credentials")
	}
	var user LocalUser
	var hash string
	var disabled bool
	err := pool.QueryRow(ctx, `
		SELECT id, username, COALESCE(email, ''), role, password_hash, disabled
		FROM app_users WHERE LOWER(username) = LOWER($1)
	`, username).Scan(&user.ID, &user.Username, &user.Email, &user.Role, &hash, &disabled)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Keep unknown-user failures on the same Argon2 path.
			_, _ = HashPassword("invalid-password-value")
			return nil, fmt.Errorf("invalid credentials")
		}
		return nil, fmt.Errorf("load local user: %w", err)
	}
	if disabled || !VerifyPassword(hash, password) {
		return nil, fmt.Errorf("invalid credentials")
	}
	return &user, nil
}

// CreateLocalUser creates the bootstrap or recovery administrator. It is used
// only by the host-invoked API admin command, never by a public HTTP route.
func CreateLocalUser(ctx context.Context, pool *pgxpool.Pool, username, email, password string) (*LocalUser, error) {
	if pool == nil {
		return nil, fmt.Errorf("database pool is required")
	}
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)
	if len(username) < 3 || len(username) > 128 {
		return nil, fmt.Errorf("username must be between 3 and 128 characters")
	}
	if strings.ContainsAny(username, "\t\r\n") {
		return nil, fmt.Errorf("username contains invalid whitespace")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	user := &LocalUser{ID: uuid.New(), Username: username, Email: email, Role: "administrator"}
	_, err = pool.Exec(ctx, `
		INSERT INTO app_users (id, username, email, password_hash, role)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5)
	`, user.ID, user.Username, user.Email, hash, user.Role)
	if err != nil {
		return nil, fmt.Errorf("create local user: %w", err)
	}
	return user, nil
}

// CreateLocalSession persists a random browser session identifier.
func CreateLocalSession(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, ttl time.Duration) (uuid.UUID, time.Time, error) {
	if ttl <= 0 {
		ttl = 12 * time.Hour
	}
	id := uuid.New()
	expires := time.Now().UTC().Add(ttl)
	_, err := pool.Exec(ctx, `INSERT INTO app_web_sessions (id, user_id, expires_at) VALUES ($1, $2, $3)`, id, userID, expires)
	if err != nil {
		return uuid.Nil, time.Time{}, fmt.Errorf("create session: %w", err)
	}
	return id, expires, nil
}

// LocalSessionUser resolves a non-expired local session.
func LocalSessionUser(ctx context.Context, pool *pgxpool.Pool, sessionID uuid.UUID) (*LocalUser, error) {
	var user LocalUser
	err := pool.QueryRow(ctx, `
		SELECT u.id, u.username, COALESCE(u.email, ''), u.role
		FROM app_web_sessions s JOIN app_users u ON u.id = s.user_id
		WHERE s.id = $1 AND s.expires_at > NOW() AND u.disabled = FALSE
	`, sessionID).Scan(&user.ID, &user.Username, &user.Email, &user.Role)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("session expired or invalid")
	}
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}
	return &user, nil
}

// DeleteLocalSession removes a single browser session.
func DeleteLocalSession(ctx context.Context, pool *pgxpool.Pool, sessionID uuid.UUID) error {
	_, err := pool.Exec(ctx, `DELETE FROM app_web_sessions WHERE id = $1`, sessionID)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// FindOrCreateExternalUser maps an allow-listed external identity to the sole
// v1 Administrator role. It is intentionally separate from local passwords.
func FindOrCreateExternalUser(ctx context.Context, pool *pgxpool.Pool, provider, subject, email, name string) (*LocalUser, error) {
	if pool == nil || strings.TrimSpace(provider) == "" || strings.TrimSpace(subject) == "" {
		return nil, fmt.Errorf("external provider and subject are required")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin external identity transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var user LocalUser
	err = tx.QueryRow(ctx, `
		SELECT u.id, u.username, COALESCE(u.email, ''), u.role
		FROM app_external_identities i JOIN app_users u ON u.id = i.user_id
		WHERE i.provider = $1 AND i.subject = $2 AND u.disabled = FALSE
	`, provider, subject).Scan(&user.ID, &user.Username, &user.Email, &user.Role)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit external identity lookup: %w", err)
		}
		return &user, nil
	}
	if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("load external identity: %w", err)
	}

	identityHash := sha256.Sum256([]byte(provider + "\x00" + subject))
	username := "external-" + fmt.Sprintf("%x", identityHash[:12])
	user = LocalUser{ID: uuid.New(), Username: username, Email: strings.TrimSpace(email), Role: "administrator"}
	if _, err := tx.Exec(ctx, `
		INSERT INTO app_users (id, username, email, password_hash, role)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5)
	`, user.ID, user.Username, user.Email, "$external-identity$", user.Role); err != nil {
		return nil, fmt.Errorf("create external user: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO app_external_identities (provider, subject, user_id) VALUES ($1, $2, $3)`, provider, subject, user.ID); err != nil {
		return nil, fmt.Errorf("create external identity: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit external identity: %w", err)
	}
	_ = name // retained for a future display-name column without changing identity semantics.
	return &user, nil
}
