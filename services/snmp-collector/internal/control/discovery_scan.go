package control

import (
	"time"
)

type activeScanState struct {
	running    bool
	startedAt  time.Time
	finishedAt time.Time
	total      int
	probed     int
	err        string
}

func (s *Server) tryBeginScan(total int) error {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	if s.scanState.running {
		return newProtoError(CodeConflict, "discovery scan already running")
	}
	s.scanState = activeScanState{
		running:   true,
		startedAt: time.Now().UTC(),
		total:     total,
		probed:    0,
	}
	return nil
}

func (s *Server) updateScanProgress(probed, total int) {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	if !s.scanState.running {
		agentDebugLog("A", "discovery_scan.go:updateScanProgress", "ignored progress update, scan not running", map[string]any{
			"probed": probed,
			"total":  total,
		})
		return
	}
	agentDebugLog("A", "discovery_scan.go:updateScanProgress", "progress updated", map[string]any{
		"probed": probed,
		"total":  total,
	})
	s.scanState.probed = probed
	if total > 0 {
		s.scanState.total = total
	}
}

func (s *Server) finishScan(scanErr string) {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	s.scanState.running = false
	s.scanState.finishedAt = time.Now().UTC()
	s.scanState.err = scanErr
}

func (s *Server) scanProgressSnapshot() activeScanState {
	s.scanMu.RLock()
	defer s.scanMu.RUnlock()
	return s.scanState
}
