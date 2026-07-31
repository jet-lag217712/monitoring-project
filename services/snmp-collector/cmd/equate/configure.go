package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func runConfigure(args []string) int {
	mode := "full"
	var extra []string
	for _, arg := range args {
		switch arg {
		case "--sites":
			mode = "sites"
		case "--users":
			mode = "users"
		default:
			extra = append(extra, arg)
		}
	}
	if len(extra) != 0 {
		fmt.Fprintln(os.Stderr, "configure accepts only --sites or --users")
		return 2
	}
	deployDir, err := resolveDeployDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configure: %v\n", err)
		return 1
	}
	if err := ensureApplianceRenderedSecrets(deployDir); err != nil {
		fmt.Fprintf(os.Stderr, "configure: %v\n", err)
		return 1
	}
	if err := runSyncDBRolePasswords(deployDir); err != nil {
		fmt.Fprintf(os.Stderr, "configure: %v\n", err)
		return 1
	}
	bootstrapper, err := resolveBootstrapper(deployDir)
	if err != nil {
		return runCollectorSetup(deployDir, mode)
	}
	cmd := exec.Command(bootstrapper, "--reconfigure", "--mode", mode)
	cmd.Dir = deployDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "configure: %v\n", err)
		return 1
	}
	return 0
}

func ensureApplianceRenderedSecrets(deployDir string) error {
	const composeEnv = "/run/equate/rendered/compose.env"
	if _, err := os.Stat(composeEnv); err == nil {
		return nil
	}
	configureScript := filepath.Join(deployDir, "scripts", "configure-vm.sh")
	if _, err := os.Stat(configureScript); err != nil {
		return fmt.Errorf("missing %s and %s", composeEnv, configureScript)
	}
	cmd := exec.Command("bash", configureScript, "--bootstrap-only")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bootstrap rendered secrets: %w", err)
	}
	return nil
}

func runCollectorSetup(deployDir string, mode string) int {
	if err := ensureApplianceRenderedSecrets(deployDir); err != nil {
		fmt.Fprintf(os.Stderr, "configure: %v\n", err)
		return 1
	}
	_ = os.Remove(filepath.Join(deployDir, ".setup-complete"))
	collector, err := exec.LookPath("collector")
	if err != nil {
		fmt.Fprintf(os.Stderr, "configure: bootstrapper not found and collector not on PATH\n")
		return 1
	}
	setupArgs := []string{"setup", "-dir", deployDir, "-theme", "auto", "-profile", "appliance", "-reconfigure", mode}
	cmd := exec.Command(collector, setupArgs...)
	cmd.Dir = deployDir
	cmd.Env = append(os.Environ(), "EQUATE_SETUP_RECONFIGURE="+mode)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "configure: %v\n", err)
		return 1
	}
	return 0
}
