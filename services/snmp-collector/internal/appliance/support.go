package appliance

import (
	"archive/tar"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var sensitiveConfigKey = regexp.MustCompile(`(?i)(password|secret|token|private.?key|credential|community)`)        //nolint:gochecknoglobals
var sensitiveTextValue = regexp.MustCompile(`(?im)(password|secret|token|credential|community)(\s*[:=]\s*)[^\s,]+`) //nolint:gochecknoglobals

// CreateSupportBundle writes a redacted support archive outside the immutable
// release tree. zstd is part of the appliance OS image, not a Go dependency.
func (l Layout) CreateSupportBundle(destination string) (string, error) {
	if err := l.Ensure(); err != nil {
		return "", err
	}
	if destination == "" {
		destination = filepath.Join(l.Backups, "equate-support-"+time.Now().UTC().Format("2006-01-02")+".tar.zst")
	}
	if !strings.HasSuffix(destination, ".tar.zst") {
		return "", fmt.Errorf("support bundle must end in .tar.zst")
	}
	tmp, err := os.CreateTemp(filepath.Dir(destination), ".equate-support-*.tar")
	if err != nil {
		return "", fmt.Errorf("create support archive: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) //nolint:errcheck
	tw := tar.NewWriter(tmp)
	entries := l.supportEntries()
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		data := entries[name]
		if err := addSupportEntry(tw, name, data); err != nil {
			_ = tw.Close()
			_ = tmp.Close()
			return "", err
		}
	}
	if err := tw.Close(); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("close support archive: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("close support archive: %w", err)
	}
	cmd := exec.Command("zstd", "--quiet", "--force", "--output", destination, tmpName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("compress support bundle: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if err := os.Chmod(destination, 0o600); err != nil {
		return "", fmt.Errorf("set support bundle mode: %w", err)
	}
	return destination, nil
}

func (l Layout) supportEntries() map[string][]byte {
	entries := map[string][]byte{
		"appliance/status.txt": []byte(l.supportStatus()),
	}
	for name, path := range map[string]string{"configuration/application.yaml": l.ApplicationYML} {
		if data, err := os.ReadFile(path); err == nil {
			entries[name] = RedactConfiguration(data)
		}
	}
	for _, spec := range []struct {
		name string
		args []string
	}{
		{name: "diagnostics/docker-version.txt", args: []string{"version", "--format", "{{.Server.Version}}"}},
		{name: "diagnostics/docker-status.txt", args: []string{"info", "--format", "{{.OperatingSystem}}"}},
	} {
		cmd := exec.Command("docker", spec.args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			out = append(out, []byte("\ncommand failed: "+err.Error()+"\n")...)
		}
		entries[spec.name] = out
	}
	if current, err := l.CurrentRelease(); err == nil {
		cmd := exec.Command("docker", "compose", "--project-name", "equate", "--env-file", filepath.Join(l.Rendered, "compose.env"), "--file", filepath.Join(current, "compose.yaml"), "logs", "--no-color", "--tail", "500")
		out, commandErr := cmd.CombinedOutput()
		if commandErr != nil {
			out = append(out, []byte("\ncommand failed: "+commandErr.Error()+"\n")...)
		}
		entries["logs/services.txt"] = RedactText(out)
	}
	for _, spec := range []struct {
		name string
		args []string
	}{
		{name: "diagnostics/disk-usage.txt", args: []string{"-h", l.Data}},
		{name: "diagnostics/memory.txt", args: []string{"-h"}},
		{name: "diagnostics/cpu.txt", args: []string{"-c", "nproc 2>/dev/null || getconf _NPROCESSORS_ONLN"}},
	} {
		command := "df"
		if spec.name == "diagnostics/memory.txt" {
			command = "free"
		}
		if spec.name == "diagnostics/cpu.txt" {
			command = "sh"
		}
		cmd := exec.Command(command, spec.args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			out = append(out, []byte("\ncommand failed: "+err.Error()+"\n")...)
		}
		entries[spec.name] = out
	}
	return entries
}

func (l Layout) supportStatus() string {
	current, err := l.CurrentRelease()
	if err != nil {
		current = "unconfigured"
	}
	return fmt.Sprintf("generated_at=%s\ncurrent_release=%s\nrelease_root=%s\n", time.Now().UTC().Format(time.RFC3339), current, l.Releases)
}

func addSupportEntry(tw *tar.Writer, name string, data []byte) error {
	header := &tar.Header{Name: name, Mode: 0o600, Size: int64(len(data)), ModTime: time.Now().UTC(), Typeflag: tar.TypeReg}
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("write support header: %w", err)
	}
	if _, err := tw.Write(data); err != nil {
		return fmt.Errorf("write support data: %w", err)
	}
	return nil
}

// RedactConfiguration recursively replaces sensitive YAML values. It is used
// only for support output; source configuration remains untouched.
func RedactConfiguration(data []byte) []byte {
	var value any
	if err := yaml.Unmarshal(data, &value); err != nil {
		return []byte("# unreadable configuration omitted\n")
	}
	redactValue(value)
	encoded, err := yaml.Marshal(value)
	if err != nil {
		return []byte("# configuration redaction failed\n")
	}
	return bytes.TrimSpace(encoded)
}

// RedactText removes common key/value secrets from diagnostics and service
// logs. It is intentionally conservative: ambiguous content is removed rather
// than exposing a customer credential in a support artifact.
func RedactText(data []byte) []byte {
	return sensitiveTextValue.ReplaceAll(data, []byte("$1$2[REDACTED]"))
}

func redactValue(value any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if sensitiveConfigKey.MatchString(key) {
				typed[key] = "[REDACTED]"
				continue
			}
			redactValue(child)
		}
	case []any:
		for _, child := range typed {
			redactValue(child)
		}
	}
}
