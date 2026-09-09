package setup

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const defaultPamHelperPath = "/opt/equate/scripts/manage-users.sh"

// resolvePamHelper finds the host PAM user-management helper script.
func resolvePamHelper(deployDir string) (string, error) {
	return ResolvePamHelperPath(deployDir)
}

// ResolvePamHelperPath returns the manage-users helper for CLI and setup tooling.
func ResolvePamHelperPath(deployDir string) (string, error) {
	candidates := []string{
		filepath.Join(deployDir, "scripts", "manage-users.sh"),
		defaultPamHelperPath,
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("PAM user helper not found (expected %s or %s)", candidates[0], candidates[1])
}

func pamUserCreate(helper, username, password string) error {
	_, err := runPamHelperOutput(helper, "create", username, password)
	return err
}

func pamUserList(helper string) (string, error) {
	return runPamHelperOutput(helper, "list")
}

func pamHasExistingUsers(helper string) (bool, string, error) {
	body, err := pamUserList(helper)
	if err != nil {
		return false, "", err
	}
	body = strings.TrimSpace(body)
	if body == "" || strings.Contains(body, "No appliance users") {
		return false, "", nil
	}
	return true, body, nil
}

func pamUserDisable(helper, username string) error {
	_, err := runPamHelperOutput(helper, "disable", username)
	return err
}

func pamUserEnable(helper, username string) error {
	_, err := runPamHelperOutput(helper, "enable", username)
	return err
}

func pamUserDelete(helper, username string) error {
	_, err := runPamHelperOutput(helper, "delete", username)
	return err
}

func pamUserReset(helper, username, password string) error {
	_, err := runPamHelperOutput(helper, "reset-password", username, password)
	return err
}

func runPamHelperOutput(helper string, args ...string) (string, error) {
	cmd := exec.Command(helper, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("pam helper %s: %s", args[0], msg)
	}
	return stdout.String(), nil
}
