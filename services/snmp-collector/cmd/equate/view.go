package main

import (
	"fmt"
	"os"
	"os/exec"

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
	socket := spec.SocketPath(deployDir)
	collector, err := exec.LookPath("collector")
	if err != nil {
		fmt.Fprintf(os.Stderr, "view: collector binary not found on PATH\n")
		return 1
	}
	cmd := exec.Command(collector, "tui", "-socket", socket, "-theme", "auto")
	cmd.Dir = deployDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "view: %v\n", err)
		return 1
	}
	return 0
}
