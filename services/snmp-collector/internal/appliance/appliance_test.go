package appliance

import (
	"archive/tar"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecretRoundTripAndRuntimeRendering(t *testing.T) {
	layout := NewLayout(t.TempDir())
	identity, err := NewSecretIdentity()
	if err != nil {
		t.Fatalf("new identity: %v", err)
	}
	if err := layout.WriteSecret(identity, "smtp", []byte("super-secret")); err != nil {
		t.Fatalf("write secret: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(layout.Secrets, "smtp.age")); err != nil || strings.Contains(string(data), "super-secret") {
		t.Fatalf("encrypted secret leaked or failed: %v", err)
	}
	path, err := layout.RenderSecret(identity, "smtp")
	if err != nil {
		t.Fatalf("render secret: %v", err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "super-secret" {
		t.Fatalf("rendered secret = %q, %v", data, err)
	}
}

func TestActivateAndFactoryResetPreservesReleases(t *testing.T) {
	layout := NewLayout(t.TempDir())
	if err := layout.Ensure(); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	release, err := layout.ReleasePath("1.0.0")
	if err != nil {
		t.Fatalf("release path: %v", err)
	}
	if err := os.MkdirAll(release, 0o755); err != nil {
		t.Fatalf("create release: %v", err)
	}
	if err := os.WriteFile(filepath.Join(release, "compose.yaml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatalf("write compose: %v", err)
	}
	if err := layout.Activate("1.0.0"); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if err := os.WriteFile(layout.ApplicationYML, []byte("site_id: before-reset\n"), 0o640); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := layout.FactoryReset(); err != nil {
		t.Fatalf("factory reset: %v", err)
	}
	if _, err := os.Stat(filepath.Join(release, "compose.yaml")); err != nil {
		t.Fatalf("release was removed: %v", err)
	}
	if _, err := os.Stat(layout.ApplicationYML); !os.IsNotExist(err) {
		t.Fatalf("configuration remains after reset: %v", err)
	}
}

func TestRedactConfiguration(t *testing.T) {
	redacted := string(RedactConfiguration([]byte("smtp:\n  password: secret\n  host: mail.example\nsnmp_community: private\n")))
	if strings.Contains(redacted, "secret") || strings.Contains(redacted, "private") || !strings.Contains(redacted, "[REDACTED]") {
		t.Fatalf("configuration was not redacted: %s", redacted)
	}
}

func TestRedactText(t *testing.T) {
	redacted := string(RedactText([]byte("password=keep-out\ncommunity: private-value\nordinary=value\n")))
	if strings.Contains(redacted, "keep-out") || strings.Contains(redacted, "private-value") || !strings.Contains(redacted, "ordinary=value") {
		t.Fatalf("text was not safely redacted: %s", redacted)
	}
}

func TestVerifyAndStageSignedUpdatePackage(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}
	content := []byte("services: {}\n")
	sum := sha256.Sum256(content)
	manifest := UpdateManifest{Version: "1.0.1", Files: map[string]string{"compose.yaml": hex.EncodeToString(sum[:])}}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	packagePath := filepath.Join(t.TempDir(), "equate-1.0.1.eqa")
	f, err := os.Create(packagePath)
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	tw := tar.NewWriter(f)
	for _, entry := range []struct {
		name string
		body []byte
	}{
		{manifestPath, manifestBytes},
		{signaturePath, []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, manifestBytes)))},
		{"release/compose.yaml", content},
	} {
		if err := tw.WriteHeader(&tar.Header{Name: entry.name, Mode: 0o644, Size: int64(len(entry.body))}); err != nil {
			t.Fatalf("write header: %v", err)
		}
		if _, err := tw.Write(entry.body); err != nil {
			t.Fatalf("write package: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close package: %v", err)
	}

	layout := NewLayout(t.TempDir())
	got, staged, err := layout.VerifyAndStageUpdatePackage(packagePath, publicKey)
	if err != nil {
		t.Fatalf("verify and stage: %v", err)
	}
	if got.Version != manifest.Version || staged != filepath.Join(layout.Releases, manifest.Version) {
		t.Fatalf("staged update = %+v, %s", got, staged)
	}
	defer filepath.Walk(staged, func(path string, info os.FileInfo, err error) error {
		if err == nil {
			if info.IsDir() {
				_ = os.Chmod(path, 0o755)
			} else {
				_ = os.Chmod(path, 0o644)
			}
		}
		return nil
	})
	if err := layout.Activate(manifest.Version); err != nil {
		t.Fatalf("activate update: %v", err)
	}
	if current, err := layout.CurrentRelease(); err != nil {
		t.Fatalf("current release: %v", err)
	} else if want, resolveErr := filepath.EvalSymlinks(staged); resolveErr != nil || current != want {
		t.Fatalf("current release = %q, want %q (%v)", current, want, resolveErr)
	}
}

func TestManagerSocketRejectsDestructiveRequestWithoutConfirmation(t *testing.T) {
	layout := NewLayout(t.TempDir())
	server := NewManagerServer(layout, filepath.Join(layout.ManagerDir, "manager.sock"))
	response := server.dispatch(t.Context(), ManagerRequest{Version: 1, ID: "reset", Method: "factory_reset", Params: map[string]any{}})
	if response.OK || !strings.Contains(response.Error, "explicit confirmation") {
		t.Fatalf("unexpected factory reset response: %+v", response)
	}
}
