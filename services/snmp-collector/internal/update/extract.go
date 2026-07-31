package update

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Extract unpacks a gzip'd tar (.eqa) into stagingDir. The staging directory is
// removed and recreated. Path traversal and absolute paths are rejected.
func Extract(eqaPath, stagingDir string) error {
	if stagingDir == "" {
		stagingDir = DefaultStagingDir
	}
	if err := os.RemoveAll(stagingDir); err != nil {
		return fmt.Errorf("clear staging dir: %w", err)
	}
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return fmt.Errorf("create staging dir: %w", err)
	}

	f, err := os.Open(eqaPath)
	if err != nil {
		return fmt.Errorf("open eqa: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read: %w", err)
		}
		name := filepath.Clean(hdr.Name)
		if name == "." || name == "" {
			continue
		}
		// Strip a single top-level directory if present (tar of bundle/).
		if parts := strings.SplitN(name, string(os.PathSeparator), 2); len(parts) == 2 {
			// Keep as-is; we place files relative to stagingDir using cleaned name.
		}
		if filepath.IsAbs(name) || strings.HasPrefix(name, ".."+string(os.PathSeparator)) || name == ".." {
			return fmt.Errorf("refusing unsafe path in archive: %q", hdr.Name)
		}
		target := filepath.Join(stagingDir, name)
		if !strings.HasPrefix(target, filepath.Clean(stagingDir)+string(os.PathSeparator)) && target != filepath.Clean(stagingDir) {
			return fmt.Errorf("refusing path escape in archive: %q", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := hdr.FileInfo().Mode().Perm()
			if mode == 0 {
				mode = 0o644
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		default:
			// Skip links and special files.
		}
	}

	// If the archive contained a single top-level directory, flatten it so
	// stagingDir/release.env exists (matches configure-vm.sh expectations).
	if err := flattenSingleRoot(stagingDir); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(stagingDir, "release.env")); err != nil {
		return fmt.Errorf("extracted bundle missing release.env under %s", stagingDir)
	}
	return nil
}

func flattenSingleRoot(stagingDir string) error {
	if _, err := os.Stat(filepath.Join(stagingDir, "release.env")); err == nil {
		return nil
	}
	entries, err := os.ReadDir(stagingDir)
	if err != nil {
		return err
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	if len(dirs) != 1 {
		return nil
	}
	root := filepath.Join(stagingDir, dirs[0])
	if _, err := os.Stat(filepath.Join(root, "release.env")); err != nil {
		return nil
	}
	tmp := stagingDir + ".flatten"
	_ = os.RemoveAll(tmp)
	if err := os.Rename(root, tmp); err != nil {
		return err
	}
	if err := os.RemoveAll(stagingDir); err != nil {
		return err
	}
	return os.Rename(tmp, stagingDir)
}
