package control

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"github.com/equate/ogsd/services/snmp-collector/internal/discovery"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/vendors"
)

func (s *Server) handleDiscoveryCandidatesList() (map[string]any, error) {
	if s.discovery == nil {
		return map[string]any{"candidates": []any{}}, nil
	}
	state, err := s.discovery.load()
	if err != nil {
		return nil, newProtoError(CodeInternal, err.Error())
	}
	items := make([]map[string]any, 0, len(state.Candidates))
	for _, c := range state.Candidates {
		items = append(items, map[string]any{
			"ip":               c.IP,
			"fingerprint":      c.Fingerprint,
			"detected_profile": c.DetectedProfile,
			"hostname":         c.Hostname,
			"description":      c.Description,
			"result":           string(c.Result),
			"error":            c.Error,
			"timestamp":        c.Timestamp.Format(time.RFC3339Nano),
		})
	}
	result := map[string]any{
		"candidates": items,
		"count":      len(items),
	}
	if !state.StartedAt.IsZero() {
		result["started_at"] = state.StartedAt.Format(time.RFC3339Nano)
	}
	if !state.FinishedAt.IsZero() {
		result["finished_at"] = state.FinishedAt.Format(time.RFC3339Nano)
	}
	if state.Error != "" {
		result["scan_error"] = state.Error
	}
	return result, nil
}

func (s *Server) handleDiscoveryScanStart(ctx context.Context, params map[string]any) (map[string]any, error) {
	async := false
	if raw, ok := params["async"]; ok {
		if b, ok := raw.(bool); ok {
			async = b
		}
	}

	scanner, cfg, err := s.newDiscoveryScanner()
	if err != nil {
		return nil, err
	}

	targetCount, err := discovery.TargetCount(ctx, cfg.Discovery)
	if err != nil {
		return nil, newProtoError(CodeValidationFailed, err.Error())
	}

	if async {
		if err := s.tryBeginScan(targetCount); err != nil {
			return nil, err
		}
		started := s.scanProgressSnapshot().startedAt
		go s.runDiscoveryScan(scanner)
		return map[string]any{
			"running":       true,
			"target_count":  targetCount,
			"started_at":    started.Format(time.RFC3339Nano),
		}, nil
	}

	if err := s.tryBeginScan(targetCount); err != nil {
		return nil, err
	}
	started := s.scanProgressSnapshot().startedAt

	scanCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	candidates, scanErr := scanner.Scan(scanCtx)
	state := discoveryScanState{
		StartedAt:  started,
		FinishedAt: time.Now().UTC(),
		Candidates: candidates,
	}
	if scanErr != nil {
		state.Error = scanErr.Error()
	}
	s.finishScan(state.Error)
	if s.discovery != nil {
		_ = s.discovery.save(state)
	}

	return s.discoveryScanResult(state, cfg, scanErr)
}

func (s *Server) handleDiscoveryScanProgress() (map[string]any, error) {
	snap := s.scanProgressSnapshot()
	result := map[string]any{
		"running": snap.running,
		"probed":  snap.probed,
		"total":   snap.total,
	}
	if !snap.startedAt.IsZero() {
		result["started_at"] = snap.startedAt.Format(time.RFC3339Nano)
	}
	if !snap.finishedAt.IsZero() {
		result["finished_at"] = snap.finishedAt.Format(time.RFC3339Nano)
	}
	if snap.err != "" {
		result["error"] = snap.err
	}
	return result, nil
}

func (s *Server) newDiscoveryScanner() (*discovery.Scanner, *config.Config, error) {
	cfg := s.manager.Current()
	if len(cfg.Discovery.AllowedCIDRs) == 0 {
		return nil, nil, newProtoError(CodeValidationFailed, "discovery allowed_cidrs is not configured")
	}
	community := strings.TrimSpace(cfg.DiscoveryCommunity())
	if community == "" {
		return nil, nil, newProtoError(CodeValidationFailed, "discovery community environment variable is not set")
	}

	registry := vendors.NewRegistry()
	scanner, err := discovery.New(cfg.Discovery, community, discovery.NewSNMPProber(), discovery.WithProfileDetector(func(identity discovery.Identity) string {
		matched, _ := registry.Match(identity.SysObjectID)
		if matched == nil {
			return "core"
		}
		return matched.Name()
	}), discovery.WithProbeProgress(func(probed, total int) {
		s.updateScanProgress(probed, total)
	}))
	if err != nil {
		return nil, nil, newProtoError(CodeValidationFailed, err.Error())
	}
	return scanner, cfg, nil
}

func (s *Server) runDiscoveryScan(scanner *discovery.Scanner) {
	scanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	started := s.scanProgressSnapshot().startedAt
	candidates, scanErr := scanner.Scan(scanCtx)
	state := discoveryScanState{
		StartedAt:  started,
		FinishedAt: time.Now().UTC(),
		Candidates: candidates,
	}
	if scanErr != nil {
		state.Error = scanErr.Error()
	}
	s.finishScan(state.Error)
	if s.discovery != nil {
		_ = s.discovery.save(state)
	}

	cfg := s.manager.Current()
	success := 0
	failed := 0
	var sampleError string
	for _, c := range candidates {
		if c.Result == discovery.ProbeSucceeded {
			success++
			continue
		}
		failed++
		if sampleError == "" && strings.TrimSpace(c.Error) != "" {
			sampleError = c.Error
		}
	}
	if s.log != nil {
		s.log.Info("discovery scan finished",
			"candidate_count", len(candidates),
			"success_count", success,
			"failed_count", failed,
			"cidr_count", len(cfg.Discovery.AllowedCIDRs),
			"sample_error", sampleError,
			"scan_error", state.Error,
		)
	}
	_ = s.audit.Record(AuditEntry{Action: "discovery.scan.start", Success: scanErr == nil, Revision: s.activeRevision(), Details: map[string]any{
		"candidate_count": len(candidates),
		"success_count":   success,
		"async":           true,
	}})
}

func (s *Server) discoveryScanResult(state discoveryScanState, cfg *config.Config, scanErr error) (map[string]any, error) {
	success := 0
	failed := 0
	var sampleError string
	for _, c := range state.Candidates {
		if c.Result == discovery.ProbeSucceeded {
			success++
			continue
		}
		failed++
		if sampleError == "" && strings.TrimSpace(c.Error) != "" {
			sampleError = c.Error
		}
	}
	result := map[string]any{
		"candidate_count": len(state.Candidates),
		"success_count":   success,
		"started_at":      state.StartedAt.Format(time.RFC3339Nano),
		"finished_at":     state.FinishedAt.Format(time.RFC3339Nano),
	}
	if state.Error != "" {
		result["error"] = state.Error
	}
	if s.log != nil {
		s.log.Info("discovery scan finished",
			"candidate_count", len(state.Candidates),
			"success_count", success,
			"failed_count", failed,
			"cidr_count", len(cfg.Discovery.AllowedCIDRs),
			"sample_error", sampleError,
			"scan_error", state.Error,
		)
	}
	_ = s.audit.Record(AuditEntry{Action: "discovery.scan.start", Success: scanErr == nil, Revision: s.activeRevision(), Details: map[string]any{
		"candidate_count": len(state.Candidates),
		"success_count":   success,
	}})
	return result, nil
}

func (s *Server) handleDiscoveryPolicyPrepare(params map[string]any) (map[string]any, error) {
	if err := validateDiscoveryPolicyParams(params); err != nil {
		_ = s.audit.Record(AuditEntry{Action: "discovery.policy.prepare", Success: false, Code: CodeValidationFailed, Message: err.Error()})
		return nil, err
	}
	rev := s.activeRevision()
	item, err := s.pending.put("discovery.policy", rev, cloneParams(params))
	if err != nil {
		return nil, newProtoError(CodeInternal, err.Error())
	}
	_ = s.audit.Record(AuditEntry{Action: "discovery.policy.prepare", Success: true, Revision: rev})
	return map[string]any{
		"confirm_token": item.Token,
		"revision":      item.Revision,
		"expires_at":    item.ExpiresAt.Format(time.RFC3339Nano),
	}, nil
}

func (s *Server) handleDiscoveryPolicyCommit(params map[string]any) (map[string]any, error) {
	token, _ := params["confirm_token"].(string)
	revision, _ := params["revision"].(string)
	if token == "" || revision == "" {
		return nil, newProtoError(CodeInvalidRequest, "confirm_token and revision are required")
	}
	if revision != s.activeRevision() {
		return nil, newProtoError(CodeRevisionMismatch, "configuration revision changed since prepare")
	}
	item, pe := s.pending.take(token, revision, "discovery.policy")
	if pe != nil {
		return nil, pe
	}
	if err := s.applyDiscoveryPolicyMutation(item.Payload); err != nil {
		return nil, err
	}
	_ = s.audit.Record(AuditEntry{Action: "discovery.policy.commit", Success: true, Revision: revision})
	return map[string]any{
		"written":  true,
		"revision": revision,
		"note":     "call config.reload to activate",
	}, nil
}

func validateDiscoveryPolicyParams(params map[string]any) error {
	cidrs, err := asStringSlice(params["allowed_cidrs"])
	if err != nil || len(cidrs) == 0 {
		return newProtoError(CodeValidationFailed, "allowed_cidrs is required")
	}
	for i, cidr := range cidrs {
		if strings.TrimSpace(cidr) == "" {
			return newProtoError(CodeValidationFailed, fmt.Sprintf("allowed_cidrs[%d] is empty", i))
		}
	}
	if raw, ok := params["community_env"]; ok && raw != nil {
		name, _ := raw.(string)
		if strings.TrimSpace(name) == "" {
			return newProtoError(CodeValidationFailed, "community_env must be non-empty when set")
		}
	}
	if raw, ok := params["max_probes_per_second"]; ok && raw != nil {
		if _, err := asFloat(raw); err != nil {
			return newProtoError(CodeValidationFailed, "max_probes_per_second must be a number")
		}
	}
	if raw, ok := params["probe_burst"]; ok && raw != nil {
		if _, err := asFloat(raw); err != nil {
			return newProtoError(CodeValidationFailed, "probe_burst must be a number")
		}
	}
	return nil
}

func (s *Server) applyDiscoveryPolicyMutation(params map[string]any) error {
	cfg := s.manager.Current()
	managedPath := cfg.ManagedInventoryPath()
	if managedPath == "" {
		return newProtoError(CodeValidationFailed, "inventory.managed_path is not configured")
	}
	cidrs, err := asStringSlice(params["allowed_cidrs"])
	if err != nil {
		return newProtoError(CodeValidationFailed, err.Error())
	}

	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()

	doc, err := config.ReadManagedDocument(managedPath)
	if err != nil {
		return newProtoError(CodeValidationFailed, err.Error())
	}
	doc.Discovery.AllowedCIDRs = cidrs
	if raw, ok := params["community_env"]; ok {
		if name, ok := raw.(string); ok && strings.TrimSpace(name) != "" {
			doc.Discovery.CommunityEnv = strings.TrimSpace(name)
		}
	}
	if raw, ok := params["max_probes_per_second"]; ok && raw != nil {
		v, err := asFloat(raw)
		if err != nil {
			return newProtoError(CodeValidationFailed, err.Error())
		}
		doc.Discovery.MaxProbesPerSecond = &v
	}
	if raw, ok := params["probe_burst"]; ok && raw != nil {
		v, err := asFloat(raw)
		if err != nil {
			return newProtoError(CodeValidationFailed, err.Error())
		}
		iv := int(v)
		doc.Discovery.ProbeBurst = &iv
	}
	if raw, ok := params["max_targets"]; ok && raw != nil {
		v, err := asFloat(raw)
		if err != nil {
			return newProtoError(CodeValidationFailed, err.Error())
		}
		iv := int(v)
		doc.Discovery.MaxTargets = &iv
	}
	if err := config.WriteManagedDocument(managedPath, doc); err != nil {
		return newProtoError(CodeValidationFailed, err.Error())
	}
	return nil
}

func (s *Server) handleDiscoveryAcceptPrepare(params map[string]any) (map[string]any, error) {
	if err := validateDiscoveryAcceptParams(params); err != nil {
		return nil, err
	}
	rev := s.activeRevision()
	item, err := s.pending.put("discovery.accept", rev, cloneParams(params))
	if err != nil {
		return nil, newProtoError(CodeInternal, err.Error())
	}
	_ = s.audit.Record(AuditEntry{Action: "discovery.accept.prepare", Success: true, Revision: rev})
	return map[string]any{
		"confirm_token": item.Token,
		"revision":      item.Revision,
		"expires_at":    item.ExpiresAt.Format(time.RFC3339Nano),
	}, nil
}

func (s *Server) handleDiscoveryAcceptCommit(params map[string]any) (map[string]any, error) {
	token, _ := params["confirm_token"].(string)
	revision, _ := params["revision"].(string)
	if token == "" || revision == "" {
		return nil, newProtoError(CodeInvalidRequest, "confirm_token and revision are required")
	}
	if revision != s.activeRevision() {
		return nil, newProtoError(CodeRevisionMismatch, "configuration revision changed since prepare")
	}
	item, pe := s.pending.take(token, revision, "discovery.accept")
	if pe != nil {
		return nil, pe
	}
	if err := s.applyDiscoveryAcceptMutation(item.Payload); err != nil {
		return nil, err
	}
	_ = s.audit.Record(AuditEntry{Action: "discovery.accept.commit", Success: true, Revision: revision})
	return map[string]any{
		"written":  true,
		"revision": revision,
		"note":     "call config.reload to activate",
	}, nil
}

func validateDiscoveryAcceptParams(params map[string]any) error {
	reviews, ok := params["reviews"].([]any)
	if !ok || len(reviews) == 0 {
		return newProtoError(CodeValidationFailed, "reviews is required")
	}
	return nil
}

func (s *Server) applyDiscoveryAcceptMutation(params map[string]any) error {
	cfg := s.manager.Current()
	managedPath := cfg.ManagedInventoryPath()
	if managedPath == "" {
		return newProtoError(CodeValidationFailed, "inventory.managed_path is not configured")
	}
	reviews, err := parseReviewedCandidates(params["reviews"])
	if err != nil {
		return newProtoError(CodeValidationFailed, err.Error())
	}

	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()

	currentManaged, err := config.ReadManagedDocument(managedPath)
	if err != nil {
		return newProtoError(CodeValidationFailed, err.Error())
	}
	if err := discovery.AcceptReviewed(managedPath, currentManaged.Devices, cfg.Devices, reviews, config.WriteManagedInventory); err != nil {
		return newProtoError(CodeValidationFailed, err.Error())
	}
	return nil
}

func parseReviewedCandidates(raw any) ([]discovery.ReviewedCandidate, error) {
	list, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("reviews must be an array")
	}
	out := make([]discovery.ReviewedCandidate, 0, len(list))
	for i, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("reviews[%d] must be an object", i)
		}
		review := discovery.ReviewedCandidate{Approved: true}
		if approved, ok := m["approved"].(bool); ok {
			review.Approved = approved
		}
		if cand, ok := m["candidate"].(map[string]any); ok {
			review.Candidate.IP = fmt.Sprint(cand["ip"])
			review.Candidate.Fingerprint = fmt.Sprint(cand["fingerprint"])
			review.Candidate.DetectedProfile = fmt.Sprint(cand["detected_profile"])
			review.Candidate.Hostname = fmt.Sprint(cand["hostname"])
			review.Candidate.Description = fmt.Sprint(cand["description"])
			review.Candidate.Result = discovery.ProbeResult(fmt.Sprint(cand["result"]))
		}
		if dev, ok := m["device"].(map[string]any); ok {
			review.Device.ID = fmt.Sprint(dev["id"])
			review.Device.Host = fmt.Sprint(dev["host"])
			if port, err := asFloat(dev["port"]); err == nil {
				review.Device.Port = uint16(port)
			}
			review.Device.CommunityEnv = fmt.Sprint(dev["community_env"])
			review.Device.Version = fmt.Sprint(dev["version"])
			if review.Device.Version == "" {
				review.Device.Version = "2c"
			}
		}
		out = append(out, review)
	}
	return out, nil
}
