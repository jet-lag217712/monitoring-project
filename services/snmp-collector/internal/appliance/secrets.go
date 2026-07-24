package appliance

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
)

// NewSecretIdentity creates the per-appliance age identity. Image provisioning
// encrypts this value with systemd-creds before persisting it under /etc.
func NewSecretIdentity() (*age.X25519Identity, error) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, fmt.Errorf("generate age identity: %w", err)
	}
	return identity, nil
}

// ParseSecretIdentity parses a runtime identity rendered by equate-init.
func ParseSecretIdentity(raw []byte) (*age.X25519Identity, error) {
	identity, err := age.ParseX25519Identity(strings.TrimSpace(string(raw)))
	if err != nil {
		return nil, fmt.Errorf("parse age identity: %w", err)
	}
	return identity, nil
}

// EncryptSecret returns an age-encrypted payload for a single appliance.
func EncryptSecret(identity *age.X25519Identity, plaintext []byte) ([]byte, error) {
	if identity == nil {
		return nil, fmt.Errorf("secret identity is required")
	}
	var out bytes.Buffer
	w, err := age.Encrypt(&out, identity.Recipient())
	if err != nil {
		return nil, fmt.Errorf("create age encryption stream: %w", err)
	}
	if _, err := w.Write(plaintext); err != nil {
		return nil, fmt.Errorf("encrypt secret: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("finalize encrypted secret: %w", err)
	}
	return out.Bytes(), nil
}

// DecryptSecret decrypts a secret only into caller-owned memory.
func DecryptSecret(identity *age.X25519Identity, ciphertext []byte) ([]byte, error) {
	if identity == nil {
		return nil, fmt.Errorf("secret identity is required")
	}
	r, err := age.Decrypt(bytes.NewReader(ciphertext), identity)
	if err != nil {
		return nil, fmt.Errorf("decrypt secret: %w", err)
	}
	plain, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read decrypted secret: %w", err)
	}
	return plain, nil
}

// WriteSecret encrypts an explicit secret name. Path traversal and extension
// ambiguity are rejected so the store cannot overwrite arbitrary host files.
func (l Layout) WriteSecret(identity *age.X25519Identity, name string, plaintext []byte) error {
	if !validSecretName(name) {
		return fmt.Errorf("invalid secret name")
	}
	if err := l.Ensure(); err != nil {
		return err
	}
	ciphertext, err := EncryptSecret(identity, plaintext)
	if err != nil {
		return err
	}
	return AtomicWriteFile(filepath.Join(l.Secrets, name+".age"), ciphertext, 0o600)
}

// RenderSecret decrypts a selected secret into the tmpfs runtime directory.
// Containers receive only this rendered file as a read-only bind mount.
func (l Layout) RenderSecret(identity *age.X25519Identity, name string) (string, error) {
	if !validSecretName(name) {
		return "", fmt.Errorf("invalid secret name")
	}
	ciphertext, err := os.ReadFile(filepath.Join(l.Secrets, name+".age"))
	if err != nil {
		return "", fmt.Errorf("read encrypted secret: %w", err)
	}
	plaintext, err := DecryptSecret(identity, ciphertext)
	if err != nil {
		return "", err
	}
	if err := l.Ensure(); err != nil {
		return "", err
	}
	path := filepath.Join(l.Rendered, name)
	if err := AtomicWriteFile(path, plaintext, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func validSecretName(name string) bool {
	if name == "" || filepath.Base(name) != name {
		return false
	}
	for _, r := range name {
		if !(r == '-' || r == '_' || r == '.' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}
