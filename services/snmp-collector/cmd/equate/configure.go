package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func runConfigure(args []string) int {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "configure accepts no arguments")
		return 2
	}
	deployDir, err := resolveDeployDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configure: %v\n", err)
		return 1
	}
	bootstrapper, err := resolveBootstrapper(deployDir)
	if err != nil {
		return runCollectorSetup(deployDir, true)
	}
	cmd := exec.Command(bootstrapper, "--reconfigure")
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

func runCollectorSetup(deployDir string, reconfigure bool) int {
	if reconfigure {
		_ = os.Remove(filepath.Join(deployDir, ".setup-complete"))
	}
	collector, err := exec.LookPath("collector")
	if err != nil {
		fmt.Fprintf(os.Stderr, "configure: bootstrapper not found and collector not on PATH\n")
		return 1
	}
	cmd := exec.Command(collector, "setup", "-dir", deployDir, "-theme", "auto", "-profile", "appliance")
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
