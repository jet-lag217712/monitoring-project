package appliance

import (
	"archive/tar"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	manifestPath   = "manifest.json"
	signaturePath  = "manifest.sig"
	maxPackageFile = 4 << 30
)

// UpdateManifest is signed as canonical JSON bytes inside a .eqa tar archive.
// Each listed file is addressed relative to the staged release directory.
type UpdateManifest struct {
	Version string            `json:"version"`
	Files   map[string]string `json:"files"`
}

// VerifyUpdatePackage verifies the same package format regardless of whether
// it came from a vendor channel, a customer mirror, or mounted virtual media.
func VerifyUpdatePackage(path string, publicKey ed25519.PublicKey) (UpdateManifest, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return UpdateManifest{}, fmt.Errorf("invalid Ed25519 public key")
	}
	manifestBytes, signature, err := readPackageMetadata(path)
	if err != nil {
		return UpdateManifest{}, err
	}
	if !ed25519.Verify(publicKey, manifestBytes, signature) {
		return UpdateManifest{}, fmt.Errorf("invalid update package signature")
	}
	var manifest UpdateManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return UpdateManifest{}, fmt.Errorf("parse update manifest: %w", err)
	}
	if !releaseName.MatchString(manifest.Version) {
		return UpdateManifest{}, fmt.Errorf("invalid manifest version %q", manifest.Version)
	}
	if len(manifest.Files) == 0 {
		return UpdateManifest{}, fmt.Errorf("update manifest has no files")
	}
	if err := verifyPackageFiles(path, manifest); err != nil {
		return UpdateManifest{}, err
	}
	return manifest, nil
}

// StageUpdatePackage extracts a previously verified package into a unique
// immutable release directory. It refuses paths outside release/.
func (l Layout) StageUpdatePackage(path string, manifest UpdateManifest) (string, error) {
	if err := l.Ensure(); err != nil {
		return "", err
	}
	final, err := l.ReleasePath(manifest.Version)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(final); err == nil {
		return "", fmt.Errorf("release %s already exists", manifest.Version)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat release destination: %w", err)
	}
	staging, err := os.MkdirTemp(l.Releases, ".stage-"+manifest.Version+"-")
	if err != nil {
		return "", fmt.Errorf("create staging directory: %w", err)
	}
	defer os.RemoveAll(staging) //nolint:errcheck

	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open update package: %w", err)
	}
	defer f.Close() //nolint:errcheck
	tr := tar.NewReader(f)
	expected := make(map[string]struct{}, len(manifest.Files))
	for name := range manifest.Files {
		expected["release/"+name] = struct{}{}
	}
	extracted := make(map[string]bool, len(expected))
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read update package: %w", err)
		}
		if h.Size < 0 || h.Size > maxPackageFile {
			return "", fmt.Errorf("invalid package file size for %s", h.Name)
		}
		if h.Name == manifestPath || h.Name == signaturePath {
			continue
		}
		if h.Typeflag != tar.TypeReg {
			return "", fmt.Errorf("package entry %q is not a regular file", h.Name)
		}
		if !strings.HasPrefix(h.Name, "release/") {
			return "", fmt.Errorf("package file %q is outside release", h.Name)
		}
		if _, ok := expected[h.Name]; !ok {
			return "", fmt.Errorf("package file %q is not listed in manifest", h.Name)
		}
		if extracted[h.Name] {
			return "", fmt.Errorf("duplicate package file %q", h.Name)
		}
		rel := strings.TrimPrefix(h.Name, "release/")
		if !safeRelativePath(rel) {
			return "", fmt.Errorf("unsafe package path %q", h.Name)
		}
		destination := filepath.Join(staging, rel)
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return "", fmt.Errorf("create package directory: %w", err)
		}
		out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return "", fmt.Errorf("create package file: %w", err)
		}
		_, copyErr := io.Copy(out, io.LimitReader(tr, maxPackageFile+1))
		closeErr := out.Close()
		if copyErr != nil || closeErr != nil {
			return "", fmt.Errorf("extract package file %s: %v %v", h.Name, copyErr, closeErr)
		}
		extracted[h.Name] = true
	}
	for name := range expected {
		if !extracted[name] {
			return "", fmt.Errorf("package missing manifest file %q", name)
		}
	}
	if _, err := os.Stat(filepath.Join(staging, "compose.yaml")); err != nil {
		return "", fmt.Errorf("staged release lacks compose.yaml: %w", err)
	}
	if err := makeReleaseReadOnly(staging); err != nil {
		return "", err
	}
	if err := os.Rename(staging, final); err != nil {
		return "", fmt.Errorf("finalize staged release: %w", err)
	}
	return final, nil
}

func makeReleaseReadOnly(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		mode := os.FileMode(0o444)
		if info.IsDir() {
			mode = 0o555
		}
		if err := os.Chmod(path, mode); err != nil {
			return fmt.Errorf("make release file immutable %s: %w", path, err)
		}
		return nil
	})
}

// VerifyAndStageUpdatePackage is the only supported staging entrypoint for
// mounted media, vendor downloads, and customer mirrors. Acquisition differs;
// signature verification and immutable staging do not.
func (l Layout) VerifyAndStageUpdatePackage(path string, publicKey ed25519.PublicKey) (UpdateManifest, string, error) {
	manifest, err := VerifyUpdatePackage(path, publicKey)
	if err != nil {
		return UpdateManifest{}, "", err
	}
	staged, err := l.StageUpdatePackage(path, manifest)
	if err != nil {
		return UpdateManifest{}, "", err
	}
	return manifest, staged, nil
}

func readPackageMetadata(path string) ([]byte, []byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open update package: %w", err)
	}
	defer f.Close() //nolint:errcheck
	tr := tar.NewReader(f)
	var manifest, signature []byte
	seenManifest := false
	seenSignature := false
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("read update package: %w", err)
		}
		if h.Size < 0 || h.Size > maxPackageFile {
			return nil, nil, fmt.Errorf("invalid package file size for %s", h.Name)
		}
		switch h.Name {
		case manifestPath:
			if seenManifest || h.Typeflag != tar.TypeReg {
				return nil, nil, fmt.Errorf("duplicate or invalid %s", manifestPath)
			}
			seenManifest = true
			manifest, err = io.ReadAll(io.LimitReader(tr, maxPackageFile+1))
		case signaturePath:
			if seenSignature || h.Typeflag != tar.TypeReg {
				return nil, nil, fmt.Errorf("duplicate or invalid %s", signaturePath)
			}
			seenSignature = true
			var raw []byte
			raw, err = io.ReadAll(io.LimitReader(tr, maxPackageFile+1))
			if err == nil {
				signature, err = base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
			}
		}
		if err != nil {
			return nil, nil, fmt.Errorf("read %s: %w", h.Name, err)
		}
	}
	if len(manifest) == 0 || len(signature) == 0 {
		return nil, nil, fmt.Errorf("update package must contain %s and %s", manifestPath, signaturePath)
	}
	return manifest, signature, nil
}

func verifyPackageFiles(path string, manifest UpdateManifest) error {
	expected := make(map[string]string, len(manifest.Files))
	for file, digest := range manifest.Files {
		if !safeRelativePath(file) || len(digest) != sha256.Size*2 {
			return fmt.Errorf("invalid manifest file %q", file)
		}
		if _, err := hex.DecodeString(digest); err != nil {
			return fmt.Errorf("invalid manifest checksum for %q", file)
		}
		expected["release/"+file] = strings.ToLower(digest)
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open update package: %w", err)
	}
	defer f.Close() //nolint:errcheck
	tr := tar.NewReader(f)
	seen := make(map[string]bool, len(expected))
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read update package: %w", err)
		}
		if h.Name == manifestPath || h.Name == signaturePath {
			continue
		}
		if h.Typeflag != tar.TypeReg || !strings.HasPrefix(h.Name, "release/") {
			return fmt.Errorf("invalid package entry %q", h.Name)
		}
		want, ok := expected[h.Name]
		if !ok {
			return fmt.Errorf("package file %q is not listed in manifest", h.Name)
		}
		if seen[h.Name] {
			return fmt.Errorf("duplicate package file %q", h.Name)
		}
		hash := sha256.New()
		if _, err := io.Copy(hash, io.LimitReader(tr, maxPackageFile+1)); err != nil {
			return fmt.Errorf("hash %s: %w", h.Name, err)
		}
		if got := hex.EncodeToString(hash.Sum(nil)); got != want {
			return fmt.Errorf("checksum mismatch for %s", h.Name)
		}
		seen[h.Name] = true
	}
	for file := range expected {
		if !seen[file] {
			return fmt.Errorf("update package missing %s", file)
		}
	}
	return nil
}

func safeRelativePath(path string) bool {
	return path != "" && !filepath.IsAbs(path) && filepath.Clean(path) == path && path != "." && !strings.HasPrefix(path, ".."+string(filepath.Separator)) && path != ".."
}
