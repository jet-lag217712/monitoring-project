package appliance

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"
)

const tlsMediaLabel = "EQUATE_TLS"

// ImportTLSMedia imports tls.crt and tls.key from a read-only removable
// volume. The input is unmounted before this function returns.
func (l Layout) ImportTLSMedia(identity *age.X25519Identity) (string, error) {
	if l.Root != "/" {
		return "", fmt.Errorf("TLS media import requires the installed appliance")
	}
	if err := l.Ensure(); err != nil {
		return "", err
	}
	device := filepath.Join("/dev/disk/by-label", tlsMediaLabel)
	if _, err := os.Stat(device); err != nil {
		return "", fmt.Errorf("attach media labeled %s: %w", tlsMediaLabel, err)
	}
	if output, err := exec.Command("mount", "-o", "ro,nosuid,nodev,noexec", device, l.TLSImport).CombinedOutput(); err != nil {
		return "", fmt.Errorf("mount TLS media: %w: %s", err, strings.TrimSpace(string(output)))
	}
	defer exec.Command("umount", l.TLSImport).Run() //nolint:errcheck
	return l.ImportTLSFiles(identity, filepath.Join(l.TLSImport, "tls.crt"), filepath.Join(l.TLSImport, "tls.key"))
}

// ImportTLSFiles validates and persists a certificate pair. It is exported for
// deterministic appliance tests; normal operator flow uses ImportTLSMedia.
func (l Layout) ImportTLSFiles(identity *age.X25519Identity, certificatePath, keyPath string) (string, error) {
	certificatePEM, err := readRegularFile(certificatePath)
	if err != nil {
		return "", fmt.Errorf("read TLS certificate: %w", err)
	}
	keyPEM, err := readRegularFile(keyPath)
	if err != nil {
		return "", fmt.Errorf("read TLS key: %w", err)
	}
	hostname, err := validateTLSCertificate(certificatePEM, keyPEM, time.Now())
	if err != nil {
		return "", err
	}
	if err := l.WriteSecret(identity, "tls.key", keyPEM); err != nil {
		return "", fmt.Errorf("encrypt TLS key: %w", err)
	}
	if err := AtomicWriteFile(l.TLSCertificate, certificatePEM, 0o644); err != nil {
		return "", fmt.Errorf("write TLS certificate: %w", err)
	}
	if err := l.setHostname(hostname); err != nil {
		return "", err
	}
	return hostname, nil
}

func readRegularFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > 128<<10 {
		return nil, fmt.Errorf("must be a non-empty regular file smaller than 128 KiB")
	}
	return os.ReadFile(path)
}

func validateTLSCertificate(certificatePEM, keyPEM []byte, now time.Time) (string, error) {
	if _, err := tls.X509KeyPair(certificatePEM, keyPEM); err != nil {
		return "", fmt.Errorf("certificate and private key do not match: %w", err)
	}
	block, _ := pem.Decode(certificatePEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return "", fmt.Errorf("TLS certificate is not PEM encoded")
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse TLS certificate: %w", err)
	}
	if now.Before(certificate.NotBefore) || !now.Before(certificate.NotAfter) {
		return "", fmt.Errorf("TLS certificate is not currently valid")
	}
	if len(certificate.DNSNames) != 1 {
		return "", fmt.Errorf("TLS certificate must contain exactly one DNS SAN")
	}
	hostname := strings.ToLower(strings.TrimSpace(certificate.DNSNames[0]))
	if strings.Contains(hostname, "*") || !strings.HasSuffix(hostname, ".equatecloud.tech") || strings.TrimSuffix(hostname, ".equatecloud.tech") == "" {
		return "", fmt.Errorf("TLS certificate SAN must be one concrete *.equatecloud.tech hostname")
	}
	return hostname, nil
}

func (l Layout) setHostname(hostname string) error {
	if err := AtomicWriteFile(filepath.Join(l.Root, "etc", "hostname"), []byte(hostname+"\n"), 0o644); err != nil {
		return fmt.Errorf("write appliance hostname: %w", err)
	}
	if l.Root != "/" {
		return nil
	}
	if output, err := exec.Command("hostnamectl", "set-hostname", hostname).CombinedOutput(); err != nil {
		return fmt.Errorf("set appliance hostname: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
