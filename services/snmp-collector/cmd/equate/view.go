package main

import (
	"fmt"
	"os"

	"github.com/equate/ogsd/services/snmp-collector/internal/tui/setup"
)

func runView(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: equate view <site-id>")
		return 2
	}
	siteID := args[0]
	deployDir, err := resolveDeployDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "view: %v\n", err)
		return 1
	}
	manifest, err := setup.LoadManifest(deployDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "view: %v\n", err)
		return 1
	}
	var spec *setup.SiteSpec
	for i := range manifest.Sites {
		if manifest.Sites[i].SiteID == siteID {
			spec = &manifest.Sites[i]
			break
		}
	}
	if spec == nil {
		fmt.Fprintf(os.Stderr, "view: site %q not found in manifest\n", siteID)
		return 1
	}
	if err := runDockerCompose(deployDir, "exec", "-it", spec.ServiceName,
		"/collector", "tui", "-socket", "/run/snmp-collector/control.sock", "-theme", "auto"); err != nil {
		fmt.Fprintf(os.Stderr, "view: %v\n", err)
		return 1
	}
	return 0
}
