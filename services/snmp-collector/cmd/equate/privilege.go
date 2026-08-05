package main

import (
	"fmt"
	"os"
	"os/exec"
)

var privilegedCommands = map[string]bool{
	"configure": true,
	"view":      true,
	"users":     true,
	"sites":     true,
	"upgrade":   true,
	"reset":     true,
}

func commandNeedsElevation(cmd string) bool {
	return privilegedCommands[cmd]
}

func runElevated(args []string) int {
	if len(args) == 0 {
		return 2
	}
	sudoArgs := append([]string{"-n", "/usr/local/bin/equate"}, args...)
	cmd := exec.Command("sudo", sudoArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "equate: sudo failed: %v\n", err)
		fmt.Fprintln(os.Stderr, "equate: appliance operators need /etc/sudoers.d/equate-appliance (re-run configure-vm.sh)")
		return 1
	}
	return 0
}
