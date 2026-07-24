package appliance

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var releaseName = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][A-Za-z0-9.-]+)?$`)

// CurrentRelease resolves the active immutable release directory.
func (l Layout) CurrentRelease() (string, error) {
	path, err := filepath.EvalSymlinks(filepath.Join(l.Releases, "current"))
	if err != nil {
		return "", fmt.Errorf("resolve current release: %w", err)
	}
	releaseRoot, err := filepath.EvalSymlinks(l.Releases)
	if err != nil {
		return "", fmt.Errorf("resolve release root: %w", err)
	}
	if filepath.Dir(path) != filepath.Clean(releaseRoot) {
		return "", fmt.Errorf("current release escapes release root")
	}
	return path, nil
}

// ReleasePath validates and returns the immutable path for version.
func (l Layout) ReleasePath(version string) (string, error) {
	if !releaseName.MatchString(version) {
		return "", fmt.Errorf("invalid release version %q", version)
	}
	return filepath.Join(l.Releases, version), nil
}

// Activate atomically points current at an already-staged release. It never
// changes the release itself, making rollback a symlink switch plus restart.
func (l Layout) Activate(version string) error {
	if err := l.Ensure(); err != nil {
		return err
	}
	target, err := l.ReleasePath(version)
	if err != nil {
		return err
	}
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("stat release %s: %w", version, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("release %s is not a directory", version)
	}
	if _, err := os.Stat(filepath.Join(target, "compose.yaml")); err != nil {
		return fmt.Errorf("release %s lacks compose.yaml: %w", version, err)
	}
	current := filepath.Join(l.Releases, "current")
	next := filepath.Join(l.Releases, ".current-next")
	_ = os.Remove(next)
	if err := os.Symlink(version, next); err != nil {
		return fmt.Errorf("create pending release link: %w", err)
	}
	if err := os.Rename(next, current); err != nil {
		return fmt.Errorf("activate release %s: %w", version, err)
	}
	return nil
}
