package config

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
)

// Manager owns the active immutable configuration snapshot.
type Manager struct {
	path    string
	current atomic.Pointer[Config]
	mu      sync.Mutex
}

// NewManager creates a configuration manager with an already validated snapshot.
func NewManager(path string, initial *Config) (*Manager, error) {
	if initial == nil {
		return nil, fmt.Errorf("initial configuration is required")
	}
	if path == "" {
		return nil, fmt.Errorf("configuration path is required")
	}
	m := &Manager{path: path}
	m.current.Store(initial)
	return m, nil
}

// Current returns the active immutable configuration snapshot.
func (m *Manager) Current() *Config {
	return m.current.Load()
}

// Reload loads, validates, and atomically activates the configured file.
func (m *Manager) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	next, err := Load(m.path)
	if err != nil {
		return err
	}
	if err := validateReload(m.Current(), next); err != nil {
		return err
	}
	m.current.Store(next)
	return nil
}

func validateReload(current, next *Config) error {
	if current == nil || next == nil {
		return fmt.Errorf("configuration snapshots must not be nil")
	}
	if current.SiteID != next.SiteID {
		return fmt.Errorf("site_id cannot change during reload")
	}
	if current.Collector.ID != next.Collector.ID {
		return fmt.Errorf("collector.id cannot change during reload")
	}
	if current.Admin != next.Admin {
		return fmt.Errorf("admin settings cannot change during reload")
	}
	if !reflect.DeepEqual(current.Publisher, next.Publisher) {
		return fmt.Errorf("publisher settings cannot change during reload")
	}
	if !reflect.DeepEqual(current.Buffer, next.Buffer) {
		return fmt.Errorf("buffer settings cannot change during reload")
	}
	if !reflect.DeepEqual(current.MQTT, next.MQTT) {
		return fmt.Errorf("mqtt settings cannot change during reload")
	}
	return nil
}
