package setup

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
)

const (
	defaultSiteCount   = 4
	minSiteCount       = 1
	maxSiteCount       = 16
	// Avoid collisions with development cloud admin ports (ingestion :9091, API :9092).
	baseAdminPort      = 19090
	manifestFile       = "sites/manifest.yaml"
	generatedComposeFile = "docker-compose.sites.generated.yml"
	collectorTemplate  = "configs/collector.yaml"
	maxSiteIDLen       = 100
)

var siteIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// SiteSpec describes one isolated collector site in a multi-site deployment.
type SiteSpec struct {
	Index            int      `yaml:"index"`
	SiteID           string   `yaml:"site_id"`
	CollectorID      string   `yaml:"collector_id"`
	MQTTClientID     string   `yaml:"mqtt_client_id"`
	ServiceName      string   `yaml:"service_name"`
	CIDR             string   `yaml:"cidr"`
	AdminPort        int      `yaml:"admin_port"`
	UpstreamSiteIDs  []string `yaml:"upstream_site_ids,omitempty"`
	HubDeviceIDs     []string `yaml:"hub_device_ids,omitempty"`
}

// Manifest is the generated multi-site deployment contract.
type Manifest struct {
	Version       int        `yaml:"version"`
	SiteCount     int        `yaml:"site_count"`
	BaseAdminPort int        `yaml:"base_admin_port"`
	ProbeRate     float64    `yaml:"probe_rate"`
	ProbeBurst    int        `yaml:"probe_burst"`
	Sites         []SiteSpec `yaml:"sites"`
}

func manifestPath(deployDir string) string {
	return filepath.Join(deployDir, manifestFile)
}

func generatedComposePath(deployDir string) string {
	return filepath.Join(deployDir, generatedComposeFile)
}

func (s SiteSpec) siteRoot(deployDir string) string {
	return filepath.Join(deployDir, "sites", s.SiteID)
}

func (s SiteSpec) ConfigPath(deployDir string) string {
	return filepath.Join(s.siteRoot(deployDir), "configs", "collector.yaml")
}

func (s SiteSpec) ManagedInventoryPath(deployDir string) string {
	return filepath.Join(s.siteRoot(deployDir), "managed", "managed-inventory.yaml")
}

func (s SiteSpec) RunDir(deployDir string) string {
	return filepath.Join(s.siteRoot(deployDir), "run")
}

func (s SiteSpec) SocketPath(deployDir string) string {
	return filepath.Join(s.RunDir(deployDir), "control.sock")
}

func (s SiteSpec) ManagedDir(deployDir string) string {
	return filepath.Join(s.siteRoot(deployDir), "managed")
}

func (s SiteSpec) VolumeName() string {
	return volumeNameForSiteID(s.SiteID)
}

func (s SiteSpec) AdminURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", s.AdminPort)
}

func defaultCIDRForIndex(index int) string {
	return fmt.Sprintf("10.255.%d.0/24", index-1)
}

func siteIDForIndex(index int) string {
	return fmt.Sprintf("site-%03d", index)
}

func collectorIDForSiteID(siteID string) string {
	return collectorIDForProfile(ProfileConfigFor(ProfileDevVxrail), siteID)
}

func mqttClientIDForSiteID(siteID string) string {
	return mqttClientIDForProfile(ProfileConfigFor(ProfileDevVxrail), siteID)
}

// dockerResourceSlug lowercases site IDs for Docker service/image/volume names.
func dockerResourceSlug(siteID string) string {
	return strings.ToLower(siteID)
}

func serviceNameForSiteID(siteID string) string {
	return "snmp-collector-" + dockerResourceSlug(siteID)
}

func volumeNameForSiteID(siteID string) string {
	return "collector-state-" + dockerResourceSlug(siteID)
}

// BuildSiteSpecs derives per-site identities and ports from operator-provided IDs and CIDRs.
func BuildSiteSpecs(profile Profile, count int, siteIDs, cidrs []string) ([]SiteSpec, error) {
	cfg := ProfileConfigFor(profile)
	if count < minSiteCount || count > maxSiteCount {
		return nil, fmt.Errorf("site count must be between %d and %d", minSiteCount, maxSiteCount)
	}
	if len(siteIDs) != count {
		return nil, fmt.Errorf("expected %d site id values, got %d", count, len(siteIDs))
	}
	if len(cidrs) != count {
		return nil, fmt.Errorf("expected %d CIDR values, got %d", count, len(cidrs))
	}
	specs := make([]SiteSpec, 0, count)
	seenSiteID := make(map[string]struct{}, count)
	seenSiteIDLower := make(map[string]string, count)
	seenCIDR := make(map[string]struct{}, count)
	for i := 1; i <= count; i++ {
		siteID := strings.TrimSpace(siteIDs[i-1])
		if err := validateSiteID(siteID); err != nil {
			return nil, fmt.Errorf("site %d: %w", i, err)
		}
		if _, ok := seenSiteID[siteID]; ok {
			return nil, fmt.Errorf("site %d: duplicate site id %q", i, siteID)
		}
		lowerSiteID := strings.ToLower(siteID)
		if orig, ok := seenSiteIDLower[lowerSiteID]; ok {
			return nil, fmt.Errorf("site %d: site id %q conflicts with %q (case-insensitive)", i, siteID, orig)
		}
		seenSiteID[siteID] = struct{}{}
		seenSiteIDLower[lowerSiteID] = siteID
		cidr := strings.TrimSpace(cidrs[i-1])
		if err := validateCIDR(cidr); err != nil {
			return nil, fmt.Errorf("site %d: %w", i, err)
		}
		if _, ok := seenCIDR[cidr]; ok {
			return nil, fmt.Errorf("site %d: duplicate CIDR %q", i, cidr)
		}
		seenCIDR[cidr] = struct{}{}
		spec := SiteSpec{
			Index:        i,
			SiteID:       siteID,
			CollectorID:  collectorIDForProfile(cfg, siteID),
			MQTTClientID: mqttClientIDForProfile(cfg, siteID),
			ServiceName:  serviceNameForSiteID(siteID),
			CIDR:         cidr,
			AdminPort:    cfg.BaseAdminPort + (i - 1),
		}
		specs = append(specs, spec)
	}
	return specs, nil
}

func validateSiteID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("site id is required")
	}
	if len(id) > maxSiteIDLen {
		return fmt.Errorf("site id must be at most %d characters", maxSiteIDLen)
	}
	if !siteIDPattern.MatchString(id) {
		return fmt.Errorf("site id %q has invalid format", id)
	}
	return nil
}

func validateCIDR(cidr string) error {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return fmt.Errorf("CIDR is required")
	}
	if _, _, err := net.ParseCIDR(cidr); err != nil {
		return fmt.Errorf("invalid CIDR %q: %w", cidr, err)
	}
	return nil
}

func parseSiteCount(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultSiteCount, nil
	}
	count, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("site count must be a number")
	}
	if count < minSiteCount || count > maxSiteCount {
		return 0, fmt.Errorf("site count must be between %d and %d", minSiteCount, maxSiteCount)
	}
	return count, nil
}

// WriteManifest persists the generated site manifest.
func WriteManifest(deployDir string, manifest Manifest) error {
	path := manifestPath(deployDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	manifest.Version = 1
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadManifest reads the generated site manifest.
func LoadManifest(deployDir string) (Manifest, error) {
	path := manifestPath(deployDir)
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, err
	}
	if len(manifest.Sites) == 0 {
		return Manifest{}, fmt.Errorf("manifest has no sites")
	}
	return manifest, nil
}

type collectorTemplateDoc struct {
	SiteID       string               `yaml:"site_id"`
	Collector    config.CollectorConfig `yaml:"collector"`
	PollInterval string               `yaml:"poll_interval"`
	MaxWorkers   int                  `yaml:"max_workers"`
	Admin        config.AdminConfig   `yaml:"admin"`
	SNMP         config.SNMPConfig    `yaml:"snmp"`
	Publisher    config.PublisherConfig `yaml:"publisher"`
	Buffer       config.BufferConfig  `yaml:"buffer"`
	MQTT         config.MQTTConfig    `yaml:"mqtt"`
	Inventory    config.InventoryConfig `yaml:"inventory"`
	Discovery    config.DiscoveryConfig `yaml:"discovery"`
	Devices      []config.DeviceConfig `yaml:"devices"`
}

// WriteSiteArtifacts generates per-site static config and managed discovery seed.
func WriteSiteArtifacts(deployDir string, specs []SiteSpec, rate float64, burst int, communityEnv string) error {
	templatePath := filepath.Join(deployDir, collectorTemplate)
	templateData, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("read collector template: %w", err)
	}
	var template collectorTemplateDoc
	if err := yaml.Unmarshal(templateData, &template); err != nil {
		return fmt.Errorf("parse collector template: %w", err)
	}
	for _, spec := range specs {
		if err := os.MkdirAll(filepath.Dir(spec.ConfigPath(deployDir)), 0o750); err != nil {
			return err
		}
		if err := os.MkdirAll(spec.ManagedDir(deployDir), 0o750); err != nil {
			return err
		}
		if err := os.MkdirAll(spec.RunDir(deployDir), 0o750); err != nil {
			return err
		}
		doc := template
		doc.SiteID = spec.SiteID
		doc.Collector.ID = spec.CollectorID
		doc.MQTT.ClientID = spec.MQTTClientID
		cfgData, err := yaml.Marshal(doc)
		if err != nil {
			return err
		}
		cfgPath := spec.ConfigPath(deployDir)
		tmp := cfgPath + ".tmp"
		if err := os.WriteFile(tmp, cfgData, 0o600); err != nil {
			return err
		}
		if err := os.Rename(tmp, cfgPath); err != nil {
			return err
		}
		if err := writeSeedManaged(spec.ManagedInventoryPath(deployDir), []string{spec.CIDR}, rate, burst, communityEnv); err != nil {
			return err
		}
	}
	return nil
}

// GenerateCompose writes the per-site Docker Compose services file.
func GenerateCompose(deployDir string, profile Profile, specs []SiteSpec, buildContext string) error {
	var b strings.Builder
	b.WriteString("# Generated by collector setup — do not edit by hand.\n")
	b.WriteString("x-collector-common: &collector-common\n")
	if profile == ProfileAppliance {
		b.WriteString("  image: ${EQUATE_COLLECTOR_IMAGE:?}\n")
		b.WriteString("  pull_policy: never\n")
	} else {
		if strings.TrimSpace(buildContext) == "" {
			buildContext = "../../../services/snmp-collector"
		}
		b.WriteString("  build: ")
		b.WriteString(buildContext)
		b.WriteString("\n")
	}
	b.WriteString("  command: [\"-config\", \"/configs/collector.yaml\"]\n")
	b.WriteString("  user: \"65532:65532\"\n")
	b.WriteString("  environment:\n")
	b.WriteString("    MQTT_BROKER: ${MQTT_BROKER:?}\n")
	b.WriteString("    MQTT_PASSWORD: ${MQTT_PASSWORD:?}\n")
	b.WriteString("    SNMP_COMMUNITY: ${SNMP_COMMUNITY:?}\n")
	b.WriteString("    SNMP_DISCOVERY_COMMUNITY: ${SNMP_DISCOVERY_COMMUNITY:-${SNMP_COMMUNITY:?}}\n")
	b.WriteString("  restart: unless-stopped\n")
	if profile == ProfileAppliance {
		b.WriteString("  networks: [core, oob]\n\n")
	} else {
		b.WriteString("\n")
	}
	b.WriteString("services:\n")
	for _, spec := range specs {
		b.WriteString("  ")
		b.WriteString(spec.ServiceName)
		b.WriteString(":\n")
		b.WriteString("    <<: *collector-common\n")
		b.WriteString("    container_name: ")
		b.WriteString(spec.ServiceName)
		b.WriteString("\n")
		b.WriteString("    volumes:\n")
		b.WriteString("      - ./sites/")
		b.WriteString(spec.SiteID)
		b.WriteString("/configs/collector.yaml:/configs/collector.yaml:ro\n")
		if profile == ProfileAppliance {
			b.WriteString("      - ${EQUATE_MQTT_CA:?}:/certs/ca.crt:ro\n")
		} else {
			b.WriteString("      - ./certs/ca.crt:/certs/ca.crt:ro\n")
		}
		b.WriteString("      - ")
		b.WriteString(spec.VolumeName())
		b.WriteString(":/var/lib/snmp-collector\n")
		b.WriteString("      - ./sites/")
		b.WriteString(spec.SiteID)
		b.WriteString("/managed:/var/lib/snmp-collector/managed\n")
		b.WriteString("      - ./sites/")
		b.WriteString(spec.SiteID)
		b.WriteString("/run:/run/snmp-collector\n")
		b.WriteString("    ports:\n")
		b.WriteString("      - \"")
		b.WriteString(strconv.Itoa(spec.AdminPort))
		b.WriteString(":9090\"\n")
	}
	b.WriteString("\nvolumes:\n")
	for _, spec := range specs {
		b.WriteString("  ")
		b.WriteString(spec.VolumeName())
		b.WriteString(":\n")
	}
	path := generatedComposePath(deployDir)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func serviceNames(specs []SiteSpec) []string {
	names := make([]string, len(specs))
	for i, spec := range specs {
		names[i] = spec.ServiceName
	}
	return names
}
