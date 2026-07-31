package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/equate/ogsd/services/snmp-collector/internal/tui/setup"
)

func runUsers(args []string) int {
	if len(args) == 0 {
		usersUsage()
		return 2
	}
	switch args[0] {
	case "list":
		return runUsersList()
	case "create":
		return runUsersCreate(args[1:])
	case "delete":
		return runUsersDelete(args[1:])
	case "disable":
		return runUsersDisable(args[1:])
	case "enable":
		return runUsersEnable(args[1:])
	case "reset-password":
		return runUsersResetPassword(args[1:])
	case "help", "-h", "--help":
		usersUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown users subcommand %q\n\n", args[0])
		usersUsage()
		return 2
	}
}

func usersUsage() {
	fmt.Fprintln(os.Stderr, "Usage: equate users <command>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  list                         List appliance users")
	fmt.Fprintln(os.Stderr, "  create <username>            Create a user (prompts for password)")
	fmt.Fprintln(os.Stderr, "  delete <username>            Delete a user")
	fmt.Fprintln(os.Stderr, "  disable <username>           Disable a user")
	fmt.Fprintln(os.Stderr, "  enable <username>            Re-enable a user")
	fmt.Fprintln(os.Stderr, "  reset-password <username>    Reset a user password")
}

func runUsersList() int {
	helper, err := resolveManageUsersHelper()
	if err != nil {
		fmt.Fprintf(os.Stderr, "users: %v\n", err)
		return 1
	}
	return runManageUsers(helper, "list")
}

func runUsersCreate(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: equate users create <username>")
		return 2
	}
	password, confirm, err := promptPasswordPair("new password: ", "confirm password: ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "users: %v\n", err)
		return 1
	}
	if password != confirm {
		fmt.Fprintln(os.Stderr, "users: password confirmation does not match")
		return 1
	}
	helper, err := resolveManageUsersHelper()
	if err != nil {
		fmt.Fprintf(os.Stderr, "users: %v\n", err)
		return 1
	}
	return runManageUsers(helper, "create", args[0], password)
}

func runUsersDelete(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: equate users delete <username>")
		return 2
	}
	confirm, err := promptLine("type DELETE to confirm: ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "users: %v\n", err)
		return 1
	}
	if strings.TrimSpace(confirm) != "DELETE" {
		fmt.Fprintln(os.Stderr, "users: confirmation required (DELETE)")
		return 1
	}
	helper, err := resolveManageUsersHelper()
	if err != nil {
		fmt.Fprintf(os.Stderr, "users: %v\n", err)
		return 1
	}
	return runManageUsers(helper, "delete", args[0])
}

func runUsersDisable(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: equate users disable <username>")
		return 2
	}
	helper, err := resolveManageUsersHelper()
	if err != nil {
		fmt.Fprintf(os.Stderr, "users: %v\n", err)
		return 1
	}
	return runManageUsers(helper, "disable", args[0])
}

func runUsersEnable(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: equate users enable <username>")
		return 2
	}
	helper, err := resolveManageUsersHelper()
	if err != nil {
		fmt.Fprintf(os.Stderr, "users: %v\n", err)
		return 1
	}
	return runManageUsers(helper, "enable", args[0])
}

func runUsersResetPassword(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: equate users reset-password <username>")
		return 2
	}
	password, confirm, err := promptPasswordPair("new password: ", "confirm password: ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "users: %v\n", err)
		return 1
	}
	if password != confirm {
		fmt.Fprintln(os.Stderr, "users: password confirmation does not match")
		return 1
	}
	helper, err := resolveManageUsersHelper()
	if err != nil {
		fmt.Fprintf(os.Stderr, "users: %v\n", err)
		return 1
	}
	return runManageUsers(helper, "reset-password", args[0], password)
}

func resolveManageUsersHelper() (string, error) {
	deployDir, err := resolveDeployDir()
	if err != nil {
		return "", err
	}
	return setup.ResolvePamHelperPath(deployDir)
}

func runManageUsers(helper string, args ...string) int {
	cmd := exec.Command(helper, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "users: %v\n", err)
		return 1
	}
	return 0
}

func promptLine(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func promptPasswordPair(firstPrompt, secondPrompt string) (string, string, error) {
	first, err := promptLine(firstPrompt)
	if err != nil {
		return "", "", err
	}
	second, err := promptLine(secondPrompt)
	if err != nil {
		return "", "", err
	}
	return first, second, nil
}
