package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	appliancePostgresData = "/var/lib/equate/postgres"
	applianceMosquittoData = "/var/lib/equate/mosquitto"
)

func runReset(args []string) int {
	fs := flag.NewFlagSet("reset", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	yes := fs.Bool("yes", false, "skip interactive confirmation")
	volumes := fs.Bool("volumes", false, "remove collector state volumes")
	full := fs.Bool("full", false, "also wipe postgres data under /var/lib/equate/postgres")
	hard := fs.Bool("hard", false, "full wipe (postgres, volumes, site artifacts) and leave all containers stopped")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "reset accepts only flags")
		return 2
	}
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "reset must run as root (sudo equate reset)")
		return 1
	}

	deployDir, err := resolveDeployDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "reset: %v\n", err)
		return 1
	}

	wipePostgres := *full || *hard
	removeVolumes := *volumes || *hard
	restartAfter := !*hard

	if !*yes {
		printResetWarning(deployDir, wipePostgres, removeVolumes, restartAfter, *hard)
		fmt.Fprint(os.Stderr, "Type RESET to continue: ")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		if strings.TrimSpace(line) != "RESET" {
			fmt.Fprintln(os.Stderr, "reset cancelled")
			return 1
		}
	}

	downArgs := []string{"down", "--remove-orphans"}
	if removeVolumes {
		downArgs = append(downArgs, "--volumes")
	}
	fmt.Fprintln(os.Stderr, "stopping containers...")
	if err := runDockerCompose(deployDir, downArgs...); err != nil {
		fmt.Fprintf(os.Stderr, "reset: docker compose down: %v\n", err)
		return 1
	}

	artifacts := []string{
		filepath.Join(deployDir, ".setup-complete"),
		filepath.Join(deployDir, ".env"),
		filepath.Join(deployDir, "docker-compose.sites.generated.yml"),
		filepath.Join(deployDir, "sites"),
	}
	for _, path := range artifacts {
		if err := os.RemoveAll(path); err != nil {
			fmt.Fprintf(os.Stderr, "reset: remove %s: %v\n", path, err)
			return 1
		}
	}

	if wipePostgres {
		if err := wipeDataDir(appliancePostgresData); err != nil {
			fmt.Fprintf(os.Stderr, "reset: remove postgres data: %v\n", err)
			return 1
		}
	}
	if *hard {
		if err := wipeDataDir(applianceMosquittoData); err != nil {
			fmt.Fprintf(os.Stderr, "reset: remove mosquitto data: %v\n", err)
			return 1
		}
	}

	if !restartAfter {
		fmt.Fprintln(os.Stderr, "hard reset complete. All containers are stopped.")
		printPostHardNextSteps(deployDir)
		return 0
	}

	fmt.Fprintln(os.Stderr, "starting core stack...")
	if wipePostgres {
		if err := runDockerCompose(deployDir, "up", "-d", "postgres", "mosquitto"); err != nil {
			fmt.Fprintf(os.Stderr, "reset: start postgres: %v\n", err)
			return 1
		}
		if err := runDockerCompose(deployDir, "run", "--rm", "migrate"); err != nil {
			fmt.Fprintf(os.Stderr, "reset: database migrate: %v\n", err)
			return 1
		}
		if err := runSyncDBRolePasswords(deployDir); err != nil {
			fmt.Fprintf(os.Stderr, "reset: %v\n", err)
			return 1
		}
	}
	if err := runDockerCompose(deployDir, "up", "-d", "--remove-orphans"); err != nil {
		fmt.Fprintf(os.Stderr, "reset: start core stack: %v\n", err)
		return 1
	}
	if !wipePostgres {
		if err := runSyncDBRolePasswords(deployDir); err != nil {
			fmt.Fprintf(os.Stderr, "reset: %v\n", err)
			return 1
		}
		if err := runDockerCompose(deployDir, "up", "-d", "backend-api", "ingestion"); err != nil {
			fmt.Fprintf(os.Stderr, "reset: restart api services: %v\n", err)
			return 1
		}
	}

	fmt.Fprintln(os.Stderr, "reset complete.")
	fmt.Fprintln(os.Stderr, "Next: sudo equate configure")
	return 0
}

func printResetWarning(deployDir string, wipePostgres, removeVolumes, restartAfter, hard bool) {
	fmt.Fprintf(os.Stderr, "This stops all Equate containers and clears setup artifacts in %s.\n", deployDir)
	if removeVolumes {
		fmt.Fprintln(os.Stderr, "Named Docker volumes (collector state, postgres, mosquitto) will be removed.")
	}
	if wipePostgres {
		fmt.Fprintln(os.Stderr, "Postgres data under /var/lib/equate/postgres will be deleted.")
	}
	if hard {
		fmt.Fprintln(os.Stderr, "Mosquitto state under /var/lib/equate/mosquitto will be deleted.")
		fmt.Fprintln(os.Stderr, "Hard reset leaves the stack stopped; nothing will be restarted.")
	} else if restartAfter {
		fmt.Fprintln(os.Stderr, "The core stack will be started again after cleanup.")
	}
}

func printPostHardNextSteps(deployDir string) {
	fmt.Fprintln(os.Stderr, "Next:")
	fmt.Fprintf(os.Stderr, "  1. sudo docker compose --env-file /run/equate/rendered/compose.env -f %s/docker-compose.yml up -d\n", deployDir)
	fmt.Fprintln(os.Stderr, "  2. sudo equate configure")
}

func wipeDataDir(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	return os.MkdirAll(path, 0o750)
}
