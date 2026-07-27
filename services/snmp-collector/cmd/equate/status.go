package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/tui/setup"
)

func runStatus(args []string) int {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "status accepts no arguments")
		return 2
	}
	deployDir, err := resolveDeployDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "status: %v\n", err)
		return 1
	}

	fmt.Println("Stack services:")
	if err := runDockerCompose(deployDir, "ps", "--format", "table {{.Name}}\t{{.Status}}\t{{.Ports}}"); err != nil {
		fmt.Fprintf(os.Stderr, "status: docker compose ps: %v\n", err)
	}

	manifest, err := setup.LoadManifest(deployDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "status: manifest: %v\n", err)
	} else {
		fmt.Println("\nCollector health:")
		client := &http.Client{Timeout: 3 * time.Second}
		for _, spec := range manifest.Sites {
			url := spec.AdminURL() + "/healthz"
			status := probeURL(client, url)
			fmt.Printf("  %s (%s): %s\n", spec.SiteID, spec.ServiceName, status)
		}
	}

	coreChecks := []struct {
		name string
		url  string
	}{
		{name: "frontend", url: "http://127.0.0.1/healthz"},
		{name: "backend-api", url: "http://127.0.0.1:8000/healthz"},
		{name: "ingestion", url: "http://127.0.0.1:9091/healthz"},
	}
	client := &http.Client{Timeout: 3 * time.Second}
	fmt.Println("\nCore endpoints:")
	for _, check := range coreChecks {
		fmt.Printf("  %s: %s\n", check.name, probeURL(client, check.url))
	}
	return 0
}

func probeURL(client *http.Client, url string) string {
	resp, err := client.Get(url)
	if err != nil {
		return "unreachable (" + strings.TrimSpace(err.Error()) + ")"
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return "ok"
	}
	return fmt.Sprintf("http %d", resp.StatusCode)
}
