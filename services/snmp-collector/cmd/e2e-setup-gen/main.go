package main

import (
	"fmt"
	"os"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/tui/setup"
)

func main() {
	deployDir := "/opt/equate/current"
	if len(os.Args) > 1 {
		deployDir = os.Args[1]
	}
	profile := setup.ProfileAppliance
	cfg := setup.ProfileConfigFor(profile)
	siteIDs := []string{"site-a-mdf", "site-b-mdf"}
	cidrs := []string{"10.10.10.0/24", "10.20.20.0/24"}
	specs, err := setup.BuildSiteSpecs(profile, len(siteIDs), siteIDs, cidrs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build specs: %v\n", err)
		os.Exit(1)
	}
	manifest := setup.Manifest{
		SiteCount:     len(specs),
		BaseAdminPort: cfg.BaseAdminPort,
		ProbeRate:     5,
		ProbeBurst:    2,
		Sites:         specs,
	}
	if err := setup.WriteManifest(deployDir, manifest); err != nil {
		fmt.Fprintf(os.Stderr, "write manifest: %v\n", err)
		os.Exit(1)
	}
	if err := setup.WriteSiteArtifacts(deployDir, specs, 5, 2, "SNMP_DISCOVERY_COMMUNITY"); err != nil {
		fmt.Fprintf(os.Stderr, "write site artifacts: %v\n", err)
		os.Exit(1)
	}
	if err := setup.GenerateCompose(deployDir, profile, specs, ""); err != nil {
		fmt.Fprintf(os.Stderr, "generate compose: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(deployDir+"/.setup-complete", []byte(time.Now().UTC().Format(time.RFC3339Nano)+"\n"), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "write marker: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("generated site artifacts for", len(specs), "sites in", deployDir)
}
