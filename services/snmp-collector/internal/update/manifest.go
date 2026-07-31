package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ChannelManifest is the static JSON published to the update channel host.
type ChannelManifest struct {
	Channel  string                     `json:"channel"`
	Edition  string                     `json:"edition"`
	Latest   string                     `json:"latest"`
	Releases map[string]ReleaseManifest `json:"releases"`
}

// ReleaseManifest describes one published release.
type ReleaseManifest struct {
	PublishedAt   string                      `json:"published_at"`
	MinVersion    string                      `json:"min_version,omitempty"`
	Architectures map[string]ArtifactManifest `json:"architectures"`
}

// ArtifactManifest points at one architecture-specific .eqa.
type ArtifactManifest struct {
	Artifact  string `json:"artifact"`
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
	Signature string `json:"signature"`
}

// FetchManifest downloads and parses a channel manifest over HTTPS.
// http:// is allowed only when allowInsecureHTTP is true (local testing).
func FetchManifest(ctx context.Context, channelURL string, allowInsecureHTTP bool) (*ChannelManifest, error) {
	if strings.TrimSpace(channelURL) == "" {
		return nil, fmt.Errorf("channel URL is empty")
	}
	if err := validateFetchURL(channelURL, allowInsecureHTTP); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, channelURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create manifest request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "equate-upgrade/1")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch manifest: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var m ChannelManifest
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// Validate checks required manifest fields.
func (m *ChannelManifest) Validate() error {
	if m == nil {
		return fmt.Errorf("manifest is nil")
	}
	if strings.TrimSpace(m.Channel) == "" {
		return fmt.Errorf("manifest missing channel")
	}
	if strings.TrimSpace(m.Edition) == "" {
		return fmt.Errorf("manifest missing edition")
	}
	if strings.TrimSpace(m.Latest) == "" {
		return fmt.Errorf("manifest missing latest")
	}
	if len(m.Releases) == 0 {
		return fmt.Errorf("manifest has no releases")
	}
	if _, ok := m.Releases[m.Latest]; !ok {
		return fmt.Errorf("manifest latest %q not present in releases", m.Latest)
	}
	return nil
}

// SelectArtifact picks the .eqa for the requested edition and architecture.
func SelectArtifact(m *ChannelManifest, edition, arch string) (version string, art ArtifactManifest, err error) {
	if m == nil {
		return "", ArtifactManifest{}, fmt.Errorf("manifest is nil")
	}
	edition = strings.ToLower(strings.TrimSpace(edition))
	arch = normalizeArch(arch)
	if edition == "" {
		edition = EditionStandard
	}
	if !strings.EqualFold(m.Edition, edition) {
		return "", ArtifactManifest{}, fmt.Errorf(
			"edition mismatch: appliance is %q but channel is %q (refusing cross-edition update)",
			edition, m.Edition,
		)
	}
	rel, ok := m.Releases[m.Latest]
	if !ok {
		return "", ArtifactManifest{}, fmt.Errorf("latest release %q missing", m.Latest)
	}
	art, ok = rel.Architectures[arch]
	if !ok {
		return "", ArtifactManifest{}, fmt.Errorf("no artifact for architecture %q in release %s", arch, m.Latest)
	}
	if strings.TrimSpace(art.URL) == "" || strings.TrimSpace(art.SHA256) == "" || strings.TrimSpace(art.Signature) == "" {
		return "", ArtifactManifest{}, fmt.Errorf("artifact for %s/%s incomplete (need url, sha256, signature)", m.Latest, arch)
	}
	return m.Latest, art, nil
}

func normalizeArch(arch string) string {
	switch strings.ToLower(strings.TrimSpace(arch)) {
	case "x86_64", "amd64":
		return "amd64"
	case "aarch64", "arm64":
		return "arm64"
	default:
		return strings.ToLower(strings.TrimSpace(arch))
	}
}

func validateFetchURL(raw string, allowInsecureHTTP bool) error {
	switch {
	case strings.HasPrefix(raw, "https://"):
		return nil
	case strings.HasPrefix(raw, "http://"):
		if allowInsecureHTTP {
			return nil
		}
		return fmt.Errorf("channel URL must use HTTPS (got http://); set allowInsecureHTTP only for local testing")
	default:
		return fmt.Errorf("channel URL must be https:// (got %q)", redactURL(raw))
	}
}

// RedactURL strips query/userinfo for safe logging.
func RedactURL(raw string) string {
	if i := strings.Index(raw, "?"); i >= 0 {
		raw = raw[:i]
	}
	if at := strings.Index(raw, "@"); at >= 0 {
		if scheme := strings.Index(raw, "://"); scheme >= 0 && scheme < at {
			return raw[:scheme+3] + "<redacted>" + raw[at:]
		}
	}
	return raw
}

func redactURL(raw string) string { return RedactURL(raw) }
