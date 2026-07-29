package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func resolveConfigureScript(deployDir string) (string, error) {
	candidates := []string{
		filepath.Join(deployDir, "scripts", "configure-vm.sh"),
		"/tmp/equate-staging/configure-vm.sh",
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("configure-vm.sh not found (checked release scripts and /tmp/equate-staging)")
}

func runUpgrade(args []string) int {
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	bundle := fs.String("bundle", "", "path to staged release bundle directory")
	version := fs.String("version", "", "target release version (semver)")
	canary := fs.Bool("canary", false, "roll out collectors one site at a time with health checks")
	rollback := fs.Bool("rollback", false, "roll back to the previous release")
	yes := fs.Bool("yes", false, "skip interactive confirmation")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "upgrade accepts only flags")
		return 2
	}
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "upgrade must run as root (sudo equate upgrade)")
		return 1
	}

	deployDir, err := resolveDeployDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "upgrade: %v\n", err)
		return 1
	}
	configureScript, err := resolveConfigureScript(deployDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "upgrade: %v\n", err)
		return 1
	}

	cmdArgs := []string{"bash", configureScript}
	if *rollback {
		if !*yes {
			fmt.Fprint(os.Stderr, "This rolls back to the previous appliance release. Type ROLLBACK to continue: ")
			reader := bufio.NewReader(os.Stdin)
			line, _ := reader.ReadString('\n')
			if strings.TrimSpace(line) != "ROLLBACK" {
				fmt.Fprintln(os.Stderr, "upgrade cancelled")
				return 1
			}
		}
		cmdArgs = append(cmdArgs, "--rollback")
	} else {
		if strings.TrimSpace(*bundle) == "" || strings.TrimSpace(*version) == "" {
			fmt.Fprintln(os.Stderr, "upgrade requires --bundle and --version (or --rollback)")
			return 2
		}
		if !*yes {
			fmt.Fprintf(os.Stderr, "Upgrade appliance to version %s from bundle %s.\n", *version, *bundle)
			fmt.Fprint(os.Stderr, "Type UPGRADE to continue: ")
			reader := bufio.NewReader(os.Stdin)
			line, _ := reader.ReadString('\n')
			if strings.TrimSpace(line) != "UPGRADE" {
				fmt.Fprintln(os.Stderr, "upgrade cancelled")
				return 1
			}
		}
		cmdArgs = append(cmdArgs, "--upgrade", "--bundle", *bundle, "--version", *version)
		if *canary {
			cmdArgs = append(cmdArgs, "--canary")
		}
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "upgrade: %v\n", err)
		return 1
	}
	return 0
}
