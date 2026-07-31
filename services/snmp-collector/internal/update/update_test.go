package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		current, latest string
		want            int
	}{
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.1", 0},
		{"1.1.0", "1.0.9", 1},
		{"v1.0.0", "1.0.0", 0},
	}
	for _, tt := range tests {
		got, err := CompareVersions(tt.current, tt.latest)
		if err != nil {
			t.Fatalf("CompareVersions(%q,%q): %v", tt.current, tt.latest, err)
		}
		if got != tt.want {
			t.Fatalf("CompareVersions(%q,%q)=%d want %d", tt.current, tt.latest, got, tt.want)
		}
	}
}

func TestSelectArtifactEditionMismatch(t *testing.T) {
	m := &ChannelManifest{
		Channel: "stable",
		Edition: EditionStandard,
		Latest:  "1.0.3",
		Releases: map[string]ReleaseManifest{
			"1.0.3": {
				Architectures: map[string]ArtifactManifest{
					"amd64": {
						Artifact:  "Equate-1.0.3-amd64.eqa",
						URL:       "https://example.blob.core.windows.net/x.eqa",
						SHA256:    "abc",
						Signature: "sig",
					},
				},
			},
		},
	}
	_, _, err := SelectArtifact(m, EditionNoAuth, "amd64")
	if err == nil || !strings.Contains(err.Error(), "edition mismatch") {
		t.Fatalf("expected edition mismatch, got %v", err)
	}
}

func TestSelectArtifactArch(t *testing.T) {
	m := &ChannelManifest{
		Channel: "stable",
		Edition: EditionStandard,
		Latest:  "1.0.3",
		Releases: map[string]ReleaseManifest{
			"1.0.3": {
				Architectures: map[string]ArtifactManifest{
					"amd64": {
						Artifact:  "Equate-1.0.3-amd64.eqa",
						URL:       "https://example.blob.core.windows.net/x.eqa",
						SHA256:    "abc",
						Signature: "sig",
					},
				},
			},
		},
	}
	ver, art, err := SelectArtifact(m, EditionStandard, "x86_64")
	if err != nil {
		t.Fatal(err)
	}
	if ver != "1.0.3" || art.Artifact != "Equate-1.0.3-amd64.eqa" {
		t.Fatalf("unexpected selection: %s %#v", ver, art)
	}
}

func TestVerifyRejectsBadSHA(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "a.eqa")
	if err := os.WriteFile(path, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	sum, err := FileSHA256(path)
	if err != nil {
		t.Fatal(err)
	}
	sig := SignSHA256(priv, sum)
	if err := Verify(path, "deadbeef", sig, pub); err == nil {
		t.Fatal("expected sha256 mismatch")
	}
	if err := Verify(path, sum, sig, pub); err != nil {
		t.Fatalf("valid verify failed: %v", err)
	}
	if err := Verify(path, sum, SignSHA256(priv, "0000"), pub); err == nil {
		t.Fatal("expected signature failure")
	}
}

func TestLoadChannelConfigMissing(t *testing.T) {
	cfg, err := LoadChannelConfig(filepath.Join(t.TempDir(), "missing.conf"))
	if err != nil || cfg != nil {
		t.Fatalf("want nil,nil got %#v %v", cfg, err)
	}
}

func TestLoadChannelConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "update-channel.conf")
	content := "channel_url=https://updates.example/v1/channel/stable/manifest.json\nedition=standard\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadChannelConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ChannelURL == "" || cfg.Edition != EditionStandard {
		t.Fatalf("bad config: %#v", cfg)
	}
}

func TestCurrentVersion(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "release.env"), []byte("EQUATE_VERSION=1.2.3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := CurrentVersion(dir)
	if err != nil || v != "1.2.3" {
		t.Fatalf("got %q %v", v, err)
	}
}

func TestFetchManifestAndExtract(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	// Build a tiny .eqa in memory / on disk.
	eqaDir := t.TempDir()
	bundle := filepath.Join(eqaDir, "bundle")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "release.env"), []byte("EQUATE_VERSION=9.9.9\nEQUATE_ARCH=amd64\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	eqaPath := filepath.Join(eqaDir, "Equate-9.9.9-amd64.eqa")
	if err := writeEQA(eqaPath, bundle); err != nil {
		t.Fatal(err)
	}
	sum, err := FileSHA256(eqaPath)
	if err != nil {
		t.Fatal(err)
	}
	sig := SignSHA256(priv, sum)

	mux := http.NewServeMux()
	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		m := ChannelManifest{
			Channel: "stable",
			Edition: EditionStandard,
			Latest:  "9.9.9",
			Releases: map[string]ReleaseManifest{
				"9.9.9": {
					PublishedAt: "2026-07-31T00:00:00Z",
					Architectures: map[string]ArtifactManifest{
						"amd64": {
							Artifact:  "Equate-9.9.9-amd64.eqa",
							URL:       "http://placeholder/Equate-9.9.9-amd64.eqa",
							SHA256:    sum,
							SizeBytes: 1,
							Signature: sig,
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(m)
	})
	mux.HandleFunc("/Equate-9.9.9-amd64.eqa", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, eqaPath)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Patch artifact URL to the test server.
	man, err := FetchManifest(context.Background(), srv.URL+"/manifest.json", true)
	if err != nil {
		t.Fatal(err)
	}
	ver, art, err := SelectArtifact(man, EditionStandard, "amd64")
	if err != nil {
		t.Fatal(err)
	}
	art.URL = srv.URL + "/Equate-9.9.9-amd64.eqa"
	_ = ver

	dest := filepath.Join(t.TempDir(), "dl.eqa")
	if err := Download(context.Background(), art.URL, dest, true, nil); err != nil {
		t.Fatal(err)
	}
	if err := Verify(dest, art.SHA256, art.Signature, pub); err != nil {
		t.Fatal(err)
	}
	staging := filepath.Join(t.TempDir(), "staging")
	if err := Extract(dest, staging); err != nil {
		t.Fatal(err)
	}
	got, err := CurrentVersion(staging)
	if err != nil || got != "9.9.9" {
		t.Fatalf("extracted version %q %v", got, err)
	}
}

func writeEQA(outPath, bundleDir string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	return filepath.Walk(bundleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(bundleDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = rel
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = tw.Write(data)
		return err
	})
}

func TestValidateFetchURL(t *testing.T) {
	if err := validateFetchURL("https://ok", false); err != nil {
		t.Fatal(err)
	}
	if err := validateFetchURL("http://local", false); err == nil {
		t.Fatal("expected http rejection")
	}
	if err := validateFetchURL("http://local", true); err != nil {
		t.Fatal(err)
	}
}

func TestMeetsMinVersion(t *testing.T) {
	ok, err := MeetsMinVersion("1.0.0", "1.0.0")
	if err != nil || !ok {
		t.Fatalf("got %v %v", ok, err)
	}
	ok, err = MeetsMinVersion("0.9.0", "1.0.0")
	if err != nil || ok {
		t.Fatalf("got %v %v", ok, err)
	}
}
