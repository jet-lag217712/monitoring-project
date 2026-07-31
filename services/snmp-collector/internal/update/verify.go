package update

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// SignaturePrefix is prepended to the lowercase SHA-256 hex digest before signing.
// Changing this breaks all existing signatures.
const SignaturePrefix = "EQUATE-EQA-v1\n"

// EmbeddedPublicKeyHex is the Ed25519 public key used to verify .eqa signatures.
// Replace by regenerating keys with appliance/scripts/generate-update-keys.sh and
// updating appliance/keys/equate-updates.pub.
const EmbeddedPublicKeyHex = "b7feec191665e7cdf272a3c33bceb6f868033117140011bc5b1aa8223c284281"

// PublicKey returns the embedded Ed25519 public key.
func PublicKey() (ed25519.PublicKey, error) {
	return ParsePublicKeyHex(EmbeddedPublicKeyHex)
}

// ParsePublicKeyHex decodes a 32-byte Ed25519 public key from hex.
func ParsePublicKeyHex(hexKey string) (ed25519.PublicKey, error) {
	hexKey = strings.TrimSpace(hexKey)
	b, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}
	if len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("public key must be %d bytes, got %d", ed25519.PublicKeySize, len(b))
	}
	return ed25519.PublicKey(b), nil
}

// ParsePrivateKeyHex decodes a 64-byte Ed25519 private key from hex.
func ParsePrivateKeyHex(hexKey string) (ed25519.PrivateKey, error) {
	hexKey = strings.TrimSpace(hexKey)
	b, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}
	if len(b) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("private key must be %d bytes, got %d", ed25519.PrivateKeySize, len(b))
	}
	return ed25519.PrivateKey(b), nil
}

// FileSHA256 returns the lowercase hex SHA-256 of path.
func FileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// SigningMessage builds the canonical bytes signed for an .eqa SHA-256 hex digest.
func SigningMessage(sha256Hex string) []byte {
	return []byte(SignaturePrefix + strings.ToLower(strings.TrimSpace(sha256Hex)))
}

// SignSHA256 creates a base64 Ed25519 signature over the canonical signing message.
func SignSHA256(priv ed25519.PrivateKey, sha256Hex string) string {
	sig := ed25519.Sign(priv, SigningMessage(sha256Hex))
	return base64.StdEncoding.EncodeToString(sig)
}

// Verify checks SHA-256 and Ed25519 signature of an .eqa file.
func Verify(path, expectedSHA256, signatureB64 string, pub ed25519.PublicKey) error {
	got, err := FileSHA256(path)
	if err != nil {
		return fmt.Errorf("hash %s: %w", path, err)
	}
	if !strings.EqualFold(got, strings.TrimSpace(expectedSHA256)) {
		return fmt.Errorf("sha256 mismatch: got %s want %s", got, expectedSHA256)
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signatureB64))
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	if len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key length %d", len(pub))
	}
	if !ed25519.Verify(pub, SigningMessage(got), sig) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}
