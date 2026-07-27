package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/equate/ogsd/services/snmp-collector/internal/tui/setup"
)

func runSites(args []string) int {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "sites accepts no arguments")
		return 2
	}
	deployDir, err := resolveDeployDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sites: %v\n", err)
		return 1
	}
	manifest, err := setup.LoadManifest(deployDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sites: %v\n", err)
		return 1
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SITE_ID\tSERVICE\tADMIN_URL\tCIDR")
	for _, spec := range manifest.Sites {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", spec.SiteID, spec.ServiceName, spec.AdminURL(), spec.CIDR)
	}
	_ = w.Flush()
	return 0
}
