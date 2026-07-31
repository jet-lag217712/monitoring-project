package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func resolveDeployDir() (string, error) {
	if v := os.Getenv("EQUATE_DEPLOY_DIR"); v != "" {
		return filepath.Abs(v)
	}
	const deployDirFile = "/etc/equate/deploy-dir"
	if data, err := os.ReadFile(deployDirFile); err == nil {
		if dir := strings.TrimSpace(string(data)); dir != "" && dir != "." {
			return filepath.Abs(dir)
		}
	}
	candidates := []string{
		"/opt/equate/current",
		"/opt/equate/releases/current",
	}
	for _, dir := range candidates {
		if _, err := os.Stat(dir); err == nil {
			return filepath.Abs(dir)
		}
	}
	return "", fmt.Errorf("deploy directory not found (set EQUATE_DEPLOY_DIR or /etc/equate/deploy-dir)")
}

func resolveBootstrapper(deployDir string) (string, error) {
	candidates := []string{
		filepath.Join(deployDir, "bootstrapper.sh"),
		filepath.Join(deployDir, "scripts", "bootstrapper.sh"),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("bootstrapper not found in %s", deployDir)
}

func runDockerCompose(deployDir string, args ...string) error {
	files := dockerComposeFiles(deployDir)
	cmdArgs := []string{"compose"}
	const composeEnv = "/run/equate/rendered/compose.env"
	if _, err := os.Stat(composeEnv); err == nil {
		cmdArgs = append(cmdArgs, "--env-file", composeEnv)
	}
	for _, f := range files {
		if _, err := os.Stat(f); err == nil {
			cmdArgs = append(cmdArgs, "-f", f)
		}
	}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command("docker", cmdArgs...)
	cmd.Dir = deployDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func dockerComposeFiles(deployDir string) []string {
	return []string{
		filepath.Join(deployDir, "docker-compose.yml"),
		filepath.Join(deployDir, "docker-compose.sites.generated.yml"),
	}
}

func runSyncDBRolePasswords(deployDir string) error {
	const composeEnv = "/run/equate/rendered/compose.env"
	if _, err := os.Stat(composeEnv); err != nil {
		return nil
	}
	script := filepath.Join(deployDir, "scripts", "sync-db-role-passwords.sh")
	if _, err := os.Stat(script); err != nil {
		return fmt.Errorf("sync database role passwords: missing %s", script)
	}
	cmd := exec.Command("bash", script)
	cmd.Dir = deployDir
	cmd.Env = append(os.Environ(),
		"EQUATE_RELEASE_DIR="+deployDir,
		"EQUATE_COMPOSE_ENV="+composeEnv,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sync database role passwords: %w", err)
	}
	return nil
}
