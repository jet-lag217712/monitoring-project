package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"github.com/equate/ogsd/services/snmp-collector/internal/tui"
)

func runTUI(args []string) int {
	fs := flag.NewFlagSet("tui", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "configs/collector.example.yaml", "path to collector config file")
	socketPath := fs.String("socket", "", "control socket path (overrides config admin.control_socket)")
	themeName := fs.String("theme", "auto", "color theme: auto, light, or dark")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "tui accepts no positional arguments")
		return 2
	}

	path := *socketPath
	if path == "" {
		cfg, err := config.LoadForValidation(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load config: %v\n", err)
			return 1
		}
		path = cfg.Admin.ControlSocket
	}
	if path == "" {
		fmt.Fprintln(os.Stderr, "control socket path is required (-socket or admin.control_socket)")
		return 2
	}
	opts := tui.Options{Theme: tui.ParseThemeName(*themeName)}
	if err := tui.Run(path, opts); err != nil {
		fmt.Fprintf(os.Stderr, "tui: %v\n", err)
		return 1
	}
	return 0
}
