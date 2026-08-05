package main

import (
	"fmt"
	"os"
)

// Build metadata injected via -ldflags; defaults keep local builds explicit.
var (
	buildVersion   = "unknown"
	buildGitCommit = "unknown"
	buildTime      = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	if commandNeedsElevation(os.Args[1]) && os.Getuid() != 0 {
		os.Exit(runElevated(os.Args[1:]))
	}
	var code int
	switch os.Args[1] {
	case "configure":
		code = runConfigure(os.Args[2:])
	case "users":
		code = runUsers(os.Args[2:])
	case "view":
		code = runView(os.Args[2:])
	case "sites":
		code = runSites(os.Args[2:])
	case "status":
		code = runStatus(os.Args[2:])
	case "reset":
		code = runReset(os.Args[2:])
	case "upgrade":
		code = runUpgrade(os.Args[2:])
	case "version":
		code = runVersion(os.Args[2:])
	case "help", "-h", "--help":
		usage()
		code = 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		code = 2
	}
	os.Exit(code)
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: equate <command>\n\nCommands:\n")
	fmt.Fprintf(os.Stderr, "  configure   Run appliance setup wizard (--sites, --users, or --temperature <celsius>)\n")
	fmt.Fprintf(os.Stderr, "  users       Manage local appliance users (create, delete, list, …)\n")
	fmt.Fprintf(os.Stderr, "  reset       Stop containers and clear setup state (--hard for full wipe, no restart)\n")
	fmt.Fprintf(os.Stderr, "  upgrade     In-place upgrade (channel, --bundle, --check, --rollback)\n")
	fmt.Fprintf(os.Stderr, "  view <site> Open per-site collector operator TUI\n")
	fmt.Fprintf(os.Stderr, "  sites       List or delete configured sites (list, delete)\n")
	fmt.Fprintf(os.Stderr, "  status      Summarize stack health\n")
	fmt.Fprintf(os.Stderr, "  version     Show release version\n")
}
