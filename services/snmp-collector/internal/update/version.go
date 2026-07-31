package update

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/mod/semver"
)

// CurrentVersion reads EQUATE_VERSION from a release.env in deployDir.
func CurrentVersion(deployDir string) (string, error) {
	path := strings.TrimRight(deployDir, "/") + "/release.env"
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read release.env: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if strings.TrimSpace(key) == "EQUATE_VERSION" {
			v := strings.TrimSpace(val)
			if v == "" {
				return "", fmt.Errorf("EQUATE_VERSION empty in %s", path)
			}
			return v, nil
		}
	}
	return "", fmt.Errorf("EQUATE_VERSION not found in %s", path)
}

// CanonicalSemver prefixes a leading "v" when missing so golang.org/x/mod/semver works.
func CanonicalSemver(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}

// CompareVersions returns -1 if current < latest, 0 if equal, 1 if current > latest.
func CompareVersions(current, latest string) (int, error) {
	c := CanonicalSemver(current)
	l := CanonicalSemver(latest)
	if !semver.IsValid(c) {
		return 0, fmt.Errorf("invalid current version %q", current)
	}
	if !semver.IsValid(l) {
		return 0, fmt.Errorf("invalid latest version %q", latest)
	}
	return semver.Compare(c, l), nil
}

// UpdateAvailable reports whether latest is newer than current.
func UpdateAvailable(current, latest string) (bool, error) {
	cmp, err := CompareVersions(current, latest)
	if err != nil {
		return false, err
	}
	return cmp < 0, nil
}

// MeetsMinVersion checks current >= minVersion when minVersion is set.
func MeetsMinVersion(current, minVersion string) (bool, error) {
	minVersion = strings.TrimSpace(minVersion)
	if minVersion == "" {
		return true, nil
	}
	cmp, err := CompareVersions(current, minVersion)
	if err != nil {
		return false, err
	}
	return cmp >= 0, nil
}
