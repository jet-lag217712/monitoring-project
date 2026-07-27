package discovery

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"path/filepath"
	"strings"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"gopkg.in/yaml.v3"
)

// CandidateFile is the local, reviewable discovery file shape.
type CandidateFile struct {
	Candidates []Candidate `json:"candidates" yaml:"candidates"`
}

// ReviewFile holds operator-approved candidates ready for export or accept.
type ReviewFile struct {
	Reviews []ReviewedCandidate `json:"reviews" yaml:"reviews"`
}

// ReviewedCandidate pairs an explicitly approved result with its proposed
// secret-free inventory entry.
type ReviewedCandidate struct {
	Candidate Candidate           `json:"candidate" yaml:"candidate"`
	Device    config.DeviceConfig `json:"device" yaml:"device"`
	Approved  bool                `json:"approved" yaml:"approved"`
}

// InventoryWriter matches config.WriteManagedInventory and is injected only
// into the explicit acceptance operation.
type InventoryWriter func(path string, devices []config.DeviceConfig) error

// WriteCandidates atomically persists candidates as JSON, YAML, or YML based
// on the destination extension.
func WriteCandidates(path string, candidates []Candidate) error {
	document := CandidateFile{Candidates: append([]Candidate(nil), candidates...)}
	data, err := marshalByExtension(path, document)
	if err != nil {
		return err
	}
	if err := writeAtomic(path, data); err != nil {
		return fmt.Errorf("write candidate review file: %w", err)
	}
	return nil
}

// ReadCandidates reads a JSON or YAML candidate review file with strict fields.
func ReadCandidates(path string) ([]Candidate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read candidate review file: %w", err)
	}
	var document CandidateFile
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&document); err != nil {
			return nil, fmt.Errorf("decode candidate JSON: %w", err)
		}
		if err := ensureSingleJSONValue(decoder); err != nil {
			return nil, err
		}
	case ".yaml", ".yml":
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoder.KnownFields(true)
		if err := decoder.Decode(&document); err != nil {
			return nil, fmt.Errorf("decode candidate YAML: %w", err)
		}
		var extra any
		if err := decoder.Decode(&extra); err == nil {
			return nil, errors.New("multiple YAML documents are not supported")
		} else if !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("decode candidate YAML: %w", err)
		}
	default:
		return nil, errors.New("candidate file extension must be .json, .yaml, or .yml")
	}
	return append([]Candidate(nil), document.Candidates...), nil
}

// WriteReviews atomically persists a reviewed-candidate file.
func WriteReviews(path string, reviews []ReviewedCandidate) error {
	document := ReviewFile{Reviews: append([]ReviewedCandidate(nil), reviews...)}
	data, err := marshalByExtension(path, document)
	if err != nil {
		return err
	}
	if err := writeAtomic(path, data); err != nil {
		return fmt.Errorf("write review file: %w", err)
	}
	return nil
}

// ReadReviews reads a JSON or YAML review file with strict fields.
func ReadReviews(path string) ([]ReviewedCandidate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read review file: %w", err)
	}
	var document ReviewFile
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&document); err != nil {
			return nil, fmt.Errorf("decode review JSON: %w", err)
		}
		if err := ensureSingleJSONValue(decoder); err != nil {
			return nil, err
		}
	case ".yaml", ".yml":
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoder.KnownFields(true)
		if err := decoder.Decode(&document); err != nil {
			return nil, fmt.Errorf("decode review YAML: %w", err)
		}
		var extra any
		if err := decoder.Decode(&extra); err == nil {
			return nil, errors.New("multiple YAML documents are not supported")
		} else if !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("decode review YAML: %w", err)
		}
	default:
		return nil, errors.New("review file extension must be .json, .yaml, or .yml")
	}
	return append([]ReviewedCandidate(nil), document.Reviews...), nil
}

// ExportReviewed writes approved candidates as a standalone DeviceConfig YAML
// inventory. It does not modify active or managed inventory.
func ExportReviewed(path string, reviews []ReviewedCandidate) error {
	devices, err := approvedDevices(reviews)
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(config.ManagedInventory{Devices: devices})
	if err != nil {
		return fmt.Errorf("encode reviewed inventory: %w", err)
	}
	if err := writeAtomic(path, data); err != nil {
		return fmt.Errorf("write reviewed inventory: %w", err)
	}
	return nil
}

// AcceptReviewed explicitly appends approved candidates to managed devices
// after checking active inventory uniqueness, then invokes the supplied
// config.WriteManagedInventory-compatible workflow. It never reloads or
// interacts with the poll scheduler.
func AcceptReviewed(
	managedPath string,
	currentManaged []config.DeviceConfig,
	activeInventory []config.DeviceConfig,
	reviews []ReviewedCandidate,
	write InventoryWriter,
) error {
	if write == nil {
		return errors.New("managed inventory writer is required")
	}
	devices, err := approvedDevices(reviews)
	if err != nil {
		return err
	}
	if err := rejectActiveDuplicates(activeInventory, devices); err != nil {
		return err
	}
	devices = filterNewManagedAppends(currentManaged, devices)
	if len(devices) == 0 {
		return nil
	}

	next := make([]config.DeviceConfig, 0, len(currentManaged)+len(devices))
	next = append(next, currentManaged...)
	next = append(next, devices...)
	if err := write(managedPath, next); err != nil {
		return fmt.Errorf("accept reviewed candidates: %w", err)
	}
	return nil
}

func filterNewManagedAppends(currentManaged, proposed []config.DeviceConfig) []config.DeviceConfig {
	if len(proposed) == 0 {
		return nil
	}
	ids := make(map[string]struct{}, len(currentManaged))
	hosts := make(map[string]struct{}, len(currentManaged))
	for _, device := range currentManaged {
		if strings.TrimSpace(device.ID) != "" {
			ids[device.ID] = struct{}{}
		}
		if strings.TrimSpace(device.Host) != "" {
			hosts[canonicalHost(device.Host)] = struct{}{}
		}
	}
	filtered := make([]config.DeviceConfig, 0, len(proposed))
	for _, device := range proposed {
		if _, exists := ids[device.ID]; exists {
			continue
		}
		if _, exists := hosts[canonicalHost(device.Host)]; exists {
			continue
		}
		filtered = append(filtered, device)
	}
	return filtered
}

func approvedDevices(reviews []ReviewedCandidate) ([]config.DeviceConfig, error) {
	devices := make([]config.DeviceConfig, 0, len(reviews))
	seenIDs := make(map[string]struct{})
	seenHosts := make(map[string]struct{})
	for index, review := range reviews {
		if !review.Approved {
			continue
		}
		if review.Candidate.Result != ProbeSucceeded {
			return nil, fmt.Errorf("reviews[%d]: only successful candidates can be approved", index)
		}
		candidateIP, err := netip.ParseAddr(strings.TrimSpace(review.Candidate.IP))
		if err != nil {
			return nil, fmt.Errorf("reviews[%d]: candidate IP is invalid: %w", index, err)
		}
		device := review.Device
		deviceIP, err := netip.ParseAddr(strings.TrimSpace(device.Host))
		if err != nil || deviceIP.Unmap() != candidateIP.Unmap() {
			return nil, fmt.Errorf("reviews[%d]: device host must match candidate IP", index)
		}
		if strings.TrimSpace(device.ID) == "" {
			return nil, fmt.Errorf("reviews[%d]: device ID is required", index)
		}
		if strings.TrimSpace(device.CommunityEnv) == "" {
			return nil, fmt.Errorf("reviews[%d]: community_env is required", index)
		}
		if device.Port == 0 {
			device.Port = 161
		}
		if device.Version == "" {
			device.Version = "2c"
		}
		if device.Version != "2c" {
			return nil, fmt.Errorf("reviews[%d]: only SNMP version 2c is supported", index)
		}
		if device.Vendor == "" {
			switch review.Candidate.DetectedProfile {
			case "core", "cisco", "arista":
				device.Vendor = review.Candidate.DetectedProfile
			}
		}
		if _, duplicate := seenIDs[device.ID]; duplicate {
			return nil, fmt.Errorf("duplicate reviewed device ID %q", device.ID)
		}
		seenIDs[device.ID] = struct{}{}
		host := canonicalHost(device.Host)
		if _, duplicate := seenHosts[host]; duplicate {
			return nil, fmt.Errorf("duplicate reviewed device host %q", device.Host)
		}
		seenHosts[host] = struct{}{}
		devices = append(devices, device)
	}
	if len(devices) == 0 {
		return nil, errors.New("at least one approved candidate is required")
	}
	return devices, nil
}

func rejectActiveDuplicates(active, proposed []config.DeviceConfig) error {
	ids := make(map[string]struct{}, len(active))
	hosts := make(map[string]struct{}, len(active))
	for _, device := range active {
		ids[device.ID] = struct{}{}
		hosts[canonicalHost(device.Host)] = struct{}{}
	}
	for _, device := range proposed {
		if _, exists := ids[device.ID]; exists {
			return fmt.Errorf("device ID %q already exists in active inventory", device.ID)
		}
		if _, exists := hosts[canonicalHost(device.Host)]; exists {
			return fmt.Errorf("device host %q already exists in active inventory", device.Host)
		}
	}
	return nil
}

func marshalByExtension(path string, value any) ([]byte, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("encode candidate JSON: %w", err)
		}
		return append(data, '\n'), nil
	case ".yaml", ".yml":
		data, err := yaml.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("encode candidate YAML: %w", err)
		}
		return data, nil
	default:
		return nil, errors.New("candidate file extension must be .json, .yaml, or .yml")
	}
}

func ensureSingleJSONValue(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return errors.New("multiple JSON values are not supported")
	} else if !errors.Is(err, io.EOF) {
		return fmt.Errorf("decode candidate JSON: %w", err)
	}
	return nil
}

func canonicalHost(host string) string {
	host = strings.TrimSpace(strings.TrimSuffix(host, "."))
	if ip, err := netip.ParseAddr(host); err == nil {
		return ip.Unmap().String()
	}
	return strings.ToLower(host)
}

func writeAtomic(path string, data []byte) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("path is required")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}
	temporary, err := os.CreateTemp(dir, ".discovery-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	cleanup := true
	defer func() {
		_ = temporary.Close()
		if cleanup {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf("set file permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace destination file: %w", err)
	}
	cleanup = false

	directory, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open parent directory: %w", err)
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return fmt.Errorf("sync parent directory: %w", err)
	}
	if err := directory.Close(); err != nil {
		return fmt.Errorf("close parent directory: %w", err)
	}
	return nil
}
