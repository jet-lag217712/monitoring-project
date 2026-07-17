package control

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type pendingMutation struct {
	Token     string
	Revision  string
	Action    string
	ExpiresAt time.Time
	Payload   map[string]any
}

type pendingStore struct {
	mu    sync.Mutex
	items map[string]pendingMutation
}

func newPendingStore() *pendingStore {
	return &pendingStore{items: make(map[string]pendingMutation)}
}

func (s *pendingStore) put(action, revision string, payload map[string]any) (pendingMutation, error) {
	token, err := randomToken()
	if err != nil {
		return pendingMutation{}, err
	}
	item := pendingMutation{
		Token:     token,
		Revision:  revision,
		Action:    action,
		ExpiresAt: time.Now().UTC().Add(ConfirmTTL),
		Payload:   payload,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[token] = item
	return item, nil
}

func (s *pendingStore) take(token, revision, action string) (pendingMutation, *ProtocolError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[token]
	if !ok {
		return pendingMutation{}, newProtoError(CodeConfirmExpired, "confirm token is unknown or already used")
	}
	delete(s.items, token)
	if time.Now().UTC().After(item.ExpiresAt) {
		return pendingMutation{}, newProtoError(CodeConfirmExpired, "confirm token expired")
	}
	if item.Action != action {
		return pendingMutation{}, newProtoError(CodeInvalidRequest, "confirm token action mismatch")
	}
	if item.Revision != revision {
		return pendingMutation{}, newProtoError(CodeRevisionMismatch, "configuration revision changed since prepare")
	}
	return item, nil
}

func randomToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
