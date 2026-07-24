package appliance

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestImportTLSFilesEncryptsKeyAndSetsHostname(t *testing.T) {
	layout := NewLayout(t.TempDir())
	identity, err := NewSecretIdentity()
	if err != nil {
		t.Fatal(err)
	}
	certificate, key := testTLSMaterial(t, []string{"client-001.equatecloud.tech"})
	certPath := filepath.Join(t.TempDir(), "tls.crt")
	keyPath := filepath.Join(filepath.Dir(certPath), "tls.key")
	if err := os.WriteFile(certPath, certificate, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, key, 0o600); err != nil {
		t.Fatal(err)
	}
	hostname, err := layout.ImportTLSFiles(identity, certPath, keyPath)
	if err != nil {
		t.Fatalf("import TLS files: %v", err)
	}
	if hostname != "client-001.equatecloud.tech" {
		t.Fatalf("hostname = %q", hostname)
	}
	if data, err := os.ReadFile(filepath.Join(layout.Secrets, "tls.key.age")); err != nil || strings.Contains(string(data), string(key)) {
		t.Fatalf("TLS key was not encrypted at rest: %v", err)
	}
	if hostnameFile, err := os.ReadFile(filepath.Join(layout.Root, "etc", "hostname")); err != nil || string(hostnameFile) != hostname+"\n" {
		t.Fatalf("hostname file = %q, %v", hostnameFile, err)
	}
}

func TestImportTLSFilesRejectsMultipleSANs(t *testing.T) {
	layout := NewLayout(t.TempDir())
	identity, err := NewSecretIdentity()
	if err != nil {
		t.Fatal(err)
	}
	certificate, key := testTLSMaterial(t, []string{"one.equatecloud.tech", "two.equatecloud.tech"})
	certPath := filepath.Join(t.TempDir(), "tls.crt")
	keyPath := filepath.Join(filepath.Dir(certPath), "tls.key")
	if err := os.WriteFile(certPath, certificate, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, key, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := layout.ImportTLSFiles(identity, certPath, keyPath); err == nil {
		t.Fatal("expected multiple SAN certificate rejection")
	}
}

func testTLSMaterial(t *testing.T, dnsNames []string) ([]byte, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	der, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: dnsNames[0]},
		DNSNames:     dnsNames,
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}, &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: dnsNames[0]},
		DNSNames:     dnsNames,
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
}
