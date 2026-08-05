package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/equate/ogsd/services/snmp-collector/internal/tui/setup"
)

type configureOptions struct {
	mode        string
	temperature *float64
}

func parseConfigureArgs(args []string) (configureOptions, error) {
	opts := configureOptions{mode: "full"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--sites":
			opts.mode = "sites"
		case "--users":
			opts.mode = "users"
		case "--temperature":
			if i+1 >= len(args) {
				return configureOptions{}, fmt.Errorf("configure --temperature requires a Celsius value")
			}
			i++
			raw := strings.TrimSpace(args[i])
			v, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return configureOptions{}, fmt.Errorf("invalid temperature %q", raw)
			}
			opts.temperature = &v
		default:
			return configureOptions{}, fmt.Errorf("configure accepts only --sites, --users, or --temperature <celsius>")
		}
	}
	if opts.temperature != nil && opts.mode != "full" {
		return configureOptions{}, fmt.Errorf("configure --temperature cannot be combined with --sites or --users")
	}
	return opts, nil
}

func runConfigure(args []string) int {
	opts, err := parseConfigureArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 2
	}
	if opts.temperature != nil {
		return runConfigureTemperature(*opts.temperature)
	}
	mode := opts.mode

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

func runConfigureTemperature(temp float64) int {
	deployDir, err := resolveDeployDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configure: %v\n", err)
		return 1
	}
	manifest, err := setup.LoadManifest(deployDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "configure: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "applying %.0f°C temperature warning to %d site(s)...\n", temp, len(manifest.Sites))
	if err := setup.ApplyGlobalTemperature(deployDir, temp); err != nil {
		fmt.Fprintf(os.Stderr, "configure: %v\n", err)
		return 1
	}
	fmt.Fprintln(os.Stdout, setup.FormatTemperatureApplied(temp, len(manifest.Sites)))
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
