package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/equate/ogsd/services/snmp-collector/internal/tui"
	"github.com/equate/ogsd/services/snmp-collector/internal/tui/setup"
)

func runSetup(args []string) int {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	deployDir := fs.String("dir", ".", "deployment directory (e.g. deployments/development/vxrail)")
	themeName := fs.String("theme", "auto", "color theme: auto, light, or dark")
	profileName := fs.String("profile", "dev-vxrail", "setup profile: dev-vxrail or appliance")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "setup accepts no positional arguments")
		return 2
	}
	profile, err := setup.ParseProfile(*profileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup: %v\n", err)
		return 2
	}
	if err := setup.Run(*deployDir, tui.ParseThemeName(*themeName), buildVersion, profile); err != nil {
		fmt.Fprintf(os.Stderr, "setup: %v\n", err)
		return 1
	}
	return 0
}
