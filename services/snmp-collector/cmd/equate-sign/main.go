// Command equate-sign signs an .eqa SHA-256 digest with the Equate update Ed25519 key.
//
// Usage:
//
//	equate-sign --key /path/to/private.hex --file dist/Equate-1.0.0-amd64.eqa
//	equate-sign --key ... --sha256 <hex>
//
// The private key is a hex-encoded 64-byte Ed25519 seed+public key (see generate-update-keys.sh).
// Output is a single base64 signature line (also written to <file>.sig when --file is used).
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/equate/ogsd/services/snmp-collector/internal/update"
)

func main() {
	keyPath := flag.String("key", "", "path to hex-encoded Ed25519 private key")
	file := flag.String("file", "", "path to .eqa to hash and sign")
	sha := flag.String("sha256", "", "precomputed lowercase SHA-256 hex (alternative to --file)")
	flag.Parse()
	if strings.TrimSpace(*keyPath) == "" {
		fmt.Fprintln(os.Stderr, "equate-sign: --key is required")
		os.Exit(2)
	}
	keyHex, err := os.ReadFile(*keyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "equate-sign: read key: %v\n", err)
		os.Exit(1)
	}
	priv, err := update.ParsePrivateKeyHex(string(keyHex))
	if err != nil {
		fmt.Fprintf(os.Stderr, "equate-sign: %v\n", err)
		os.Exit(1)
	}

	sum := strings.TrimSpace(*sha)
	if sum == "" {
		if strings.TrimSpace(*file) == "" {
			fmt.Fprintln(os.Stderr, "equate-sign: --file or --sha256 required")
			os.Exit(2)
		}
		sum, err = update.FileSHA256(*file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "equate-sign: hash: %v\n", err)
			os.Exit(1)
		}
	}
	sig := update.SignSHA256(priv, sum)
	fmt.Println(sig)
	if strings.TrimSpace(*file) != "" {
		sigPath := *file + ".sig"
		if err := os.WriteFile(sigPath, []byte(sig+"\n"), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "equate-sign: write %s: %v\n", sigPath, err)
			os.Exit(1)
		}
		shaPath := *file + ".sha256"
		line := fmt.Sprintf("%s  %s\n", sum, filepathBase(*file))
		if err := os.WriteFile(shaPath, []byte(line), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "equate-sign: write %s: %v\n", shaPath, err)
			os.Exit(1)
		}
	}
}

func filepathBase(p string) string {
	i := strings.LastIndexAny(p, `/\`)
	if i < 0 {
		return p
	}
	return p[i+1:]
}
