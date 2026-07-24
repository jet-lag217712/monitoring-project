// equate is the appliance-only local and restricted-SSH administration CLI.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/equate/ogsd/services/snmp-collector/internal/appliance"
)

func main() {
	root := flag.String("root", os.Getenv("EQUATE_ROOT"), "alternate appliance root (tests/image build only)")
	flag.Parse()
	layout := appliance.NewLayout(*root)
	args := flag.Args()
	if len(args) == 0 {
		if err := appliance.RunConsole(layout); err != nil {
			fmt.Fprintln(os.Stderr, "equate:", err)
			os.Exit(1)
		}
		return
	}

	switch args[0] {
	case "snmp":
		if len(args) == 1 || (len(args) == 2 && args[1] == "tui") {
			if err := appliance.RunSNMPTUI(layout); err != nil {
				fmt.Fprintln(os.Stderr, "open SNMP TUI:", err)
				os.Exit(1)
			}
			return
		}
		if len(args) == 2 && args[1] == "reconfigure" {
			if err := appliance.RunSNMPSetup(layout); err != nil {
				fmt.Fprintln(os.Stderr, "open SNMP setup:", err)
				os.Exit(1)
			}
			return
		}
		usage()
		os.Exit(2)
	case "manager":
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		server := appliance.NewManagerServer(layout, "")
		if err := server.Listen(); err != nil {
			fmt.Fprintln(os.Stderr, "start appliance manager:", err)
			os.Exit(1)
		}
		defer server.Close() //nolint:errcheck
		if err := server.Serve(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "serve appliance manager:", err)
			os.Exit(1)
		}
	case "status":
		current, err := layout.CurrentRelease()
		if err != nil {
			current = "unconfigured"
		}
		fmt.Printf("release: %s\n", current)
	case "release":
		if len(args) != 3 || args[1] != "activate" {
			usage()
			os.Exit(2)
		}
		if err := layout.Activate(args[2]); err != nil {
			fmt.Fprintln(os.Stderr, "activate release:", err)
			os.Exit(1)
		}
		fmt.Printf("activated release %s\n", args[2])
	case "factory-reset":
		if len(args) != 2 || args[1] != "--confirm=FACTORY-RESET" {
			fmt.Fprintln(os.Stderr, "factory reset requires --confirm=FACTORY-RESET")
			os.Exit(2)
		}
		if err := layout.FactoryResetAndReboot(); err != nil {
			fmt.Fprintln(os.Stderr, "factory reset:", err)
			os.Exit(1)
		}
		fmt.Println("factory state removed; reboot to enter first-boot setup")
	case "support-bundle":
		fs := flag.NewFlagSet("support-bundle", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		output := fs.String("output", "", "destination .tar.zst path")
		if err := fs.Parse(args[1:]); err != nil {
			os.Exit(2)
		}
		path, err := layout.CreateSupportBundle(*output)
		if err != nil {
			fmt.Fprintln(os.Stderr, "generate support bundle:", err)
			os.Exit(1)
		}
		fmt.Println(path)
	case "paths":
		fmt.Println(filepath.Join(layout.Releases, "current"))
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: equate [--root path] {snmp [tui|reconfigure]|status|paths|release activate <version>|factory-reset --confirm=FACTORY-RESET|support-bundle [-output path]}")
}
