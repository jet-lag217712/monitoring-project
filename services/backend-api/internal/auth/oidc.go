package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

const googleIssuer = "https://accounts.google.com"

// Verifier validates Google ID tokens for the configured OAuth client.
type Verifier struct {
	verifier *oidc.IDTokenVerifier
}

// NewGoogleVerifier builds a JWKS-backed verifier for Google ID tokens.
func NewGoogleVerifier(ctx context.Context, clientID string) (*Verifier, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil, fmt.Errorf("google client id is required")
	}

	provider, err := oidc.NewProvider(ctx, googleIssuer)
	if err != nil {
		return nil, fmt.Errorf("oidc provider: %w", err)
	}

	return &Verifier{
		verifier: provider.Verifier(&oidc.Config{ClientID: clientID}),
	}, nil
}

// Claims are the identity fields we expose to handlers after verification.
type Claims struct {
	Subject       string
	Email         string
	Name          string
	EmailVerified bool
	HostedDomain  string
}

// Verify parses and validates a raw Google ID token.
func (v *Verifier) Verify(ctx context.Context, rawToken string) (*Claims, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil, fmt.Errorf("empty token")
	}

	idToken, err := v.verifier.Verify(ctx, rawToken)
	if err != nil {
		return nil, fmt.Errorf("verify id token: %w", err)
	}

	var payload struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		HostedDomain  string `json:"hd"`
	}
	if err := idToken.Claims(&payload); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}
	if payload.Email != "" && !payload.EmailVerified {
		return nil, fmt.Errorf("email not verified")
	}

	return &Claims{
		Subject:       idToken.Subject,
		Email:         payload.Email,
		Name:          payload.Name,
		EmailVerified: payload.EmailVerified,
		HostedDomain:  payload.HostedDomain,
	}, nil
}
