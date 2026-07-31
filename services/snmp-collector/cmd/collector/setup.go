package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/equate/ogsd/services/snmp-collector/internal/tui"
	"github.com/equate/ogsd/services/snmp-collector/internal/tui/setup"
)

func runSetup(args []string) int {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	deployDir := fs.String("dir", ".", "deployment directory (e.g. deployments/development/vxrail)")
	themeName := fs.String("theme", "auto", "color theme: auto, light, or dark")
	profileName := fs.String("profile", "dev-vxrail", "setup profile: dev-vxrail or appliance")
	reconfigureMode := fs.String("reconfigure", "", "reconfigure mode: full, sites, or users")
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
	mode := setup.ReconfigureModeFromEnv()
	if strings.TrimSpace(*reconfigureMode) != "" {
		mode, err = setup.ParseReconfigureMode(*reconfigureMode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "setup: %v\n", err)
			return 2
		}
	}
	opts := setup.RunOptions{Reconfigure: mode}
	if err := setup.Run(*deployDir, tui.ParseThemeName(*themeName), buildVersion, profile, opts); err != nil {
		fmt.Fprintf(os.Stderr, "setup: %v\n", err)
		return 1
	}
	return 0
}
