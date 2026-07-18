package control

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/discovery"
)

type discoveryScanState struct {
	StartedAt  time.Time             `json:"started_at"`
	FinishedAt time.Time             `json:"finished_at,omitempty"`
	Candidates []discovery.Candidate `json:"candidates"`
	Error      string                `json:"error,omitempty"`
}

type discoveryStore struct {
	path string
	mu   sync.RWMutex
}

func newDiscoveryStore(stateDir string) *discoveryStore {
	if stateDir == "" {
		stateDir = "."
	}
	return &discoveryStore{path: filepath.Join(stateDir, "discovery-candidates.json")}
}

func (s *discoveryStore) save(state discoveryScanState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *discoveryStore) load() (discoveryScanState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return discoveryScanState{}, nil
		}
		return discoveryScanState{}, err
	}
	var state discoveryScanState
	if err := json.Unmarshal(data, &state); err != nil {
		return discoveryScanState{}, err
	}
	return state, nil
}
