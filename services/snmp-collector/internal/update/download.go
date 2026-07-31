package update

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ProgressFunc reports download progress (bytes written, optional total).
type ProgressFunc func(written, total int64)

// Download fetches url to destPath. allowInsecureHTTP is for local testing only.
func Download(ctx context.Context, url, destPath string, allowInsecureHTTP bool, progress ProgressFunc) error {
	if err := validateFetchURL(url, allowInsecureHTTP); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create download dir: %w", err)
	}

	tmp := destPath + ".partial"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create download file: %w", err)
	}
	success := false
	defer func() {
		_ = out.Close()
		if !success {
			_ = os.Remove(tmp)
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create download request: %w", err)
	}
	req.Header.Set("User-Agent", "equate-upgrade/1")

	client := &http.Client{Timeout: 0} // large artifacts; rely on context
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", redactURL(url), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", redactURL(url), resp.StatusCode)
	}

	total := resp.ContentLength
	var written int64
	buf := make([]byte, 256*1024)
	lastReport := time.Now()
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				return fmt.Errorf("write download: %w", err)
			}
			written += int64(n)
			if progress != nil && time.Since(lastReport) > 200*time.Millisecond {
				progress(written, total)
				lastReport = time.Now()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read download: %w", readErr)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if progress != nil {
		progress(written, total)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close download: %w", err)
	}
	if err := os.Rename(tmp, destPath); err != nil {
		return fmt.Errorf("finalize download: %w", err)
	}
	success = true
	return nil
}

// ArtifactDownloadPath returns the default local path for an artifact name.
func ArtifactDownloadPath(downloadDir, artifactName string) string {
	if downloadDir == "" {
		downloadDir = DefaultDownloadDir
	}
	base := filepath.Base(strings.TrimSpace(artifactName))
	if base == "" || base == "." || base == "/" {
		base = "update.eqa"
	}
	return filepath.Join(downloadDir, base)
}
