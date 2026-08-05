package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/equate/ogsd/services/snmp-collector/internal/tui/setup"
)

func runSites(args []string) int {
	if len(args) == 0 {
		return runSitesList()
	}
	switch args[0] {
	case "list":
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, "usage: equate sites list")
			return 2
		}
		return runSitesList()
	case "delete":
		return runSitesDelete(args[1:])
	case "help", "-h", "--help":
		sitesUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown sites subcommand %q\n\n", args[0])
		sitesUsage()
		return 2
	}
}

func sitesUsage() {
	fmt.Fprintln(os.Stderr, "Usage: equate sites [command]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  list                         List configured sites from manifest (default)")
	fmt.Fprintln(os.Stderr, "  delete <site-id> [--yes]     Remove a site (collector, artifacts, DB rows)")
}

func runSitesList() int {
	deployDir, err := resolveDeployDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sites: %v\n", err)
		return 1
	}
	manifest, err := setup.LoadManifest(deployDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sites: %v\n", err)
		return 1
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SITE_ID\tSERVICE\tADMIN_URL\tCIDR")
	for _, spec := range manifest.Sites {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", spec.SiteID, spec.ServiceName, spec.AdminURL(), spec.CIDR)
	}
	_ = w.Flush()
	return 0
}

func runSitesDelete(args []string) int {
	siteID, yes, err := parseSitesDeleteArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "sites delete must run as root (sudo equate sites delete)")
		return 1
	}

	deployDir, err := resolveDeployDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sites: %v\n", err)
		return 1
	}
	manifest, err := setup.LoadManifest(deployDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sites: %v\n", err)
		return 1
	}
	updated, removed, err := setup.RemoveSiteFromManifest(manifest, siteID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sites: %v\n", err)
		return 1
	}

	if !yes {
		fmt.Fprintf(os.Stderr, "This will permanently delete site %q (collector, volume, DB rows).\n", removed.SiteID)
		confirm, err := promptLine(fmt.Sprintf("type %s to confirm: ", removed.SiteID))
		if err != nil {
			fmt.Fprintf(os.Stderr, "sites: %v\n", err)
			return 1
		}
		if confirm != removed.SiteID {
			fmt.Fprintln(os.Stderr, "sites: confirmation required")
			return 1
		}
	}

	fmt.Fprintf(os.Stderr, "stopping collector %s...\n", removed.ServiceName)
	if err := runDockerCompose(deployDir, "rm", "-sfv", removed.ServiceName); err != nil {
		fmt.Fprintf(os.Stderr, "sites: remove collector: %v\n", err)
		return 1
	}

	if err := setup.WriteManifest(deployDir, updated); err != nil {
		fmt.Fprintf(os.Stderr, "sites: write manifest: %v\n", err)
		return 1
	}
	if err := setup.GenerateCompose(deployDir, setup.ProfileAppliance, updated.Sites, ""); err != nil {
		fmt.Fprintf(os.Stderr, "sites: regenerate compose: %v\n", err)
		return 1
	}

	siteDir := filepath.Join(deployDir, "sites", removed.SiteID)
	fmt.Fprintf(os.Stderr, "removing %s...\n", siteDir)
	if err := os.RemoveAll(siteDir); err != nil {
		fmt.Fprintf(os.Stderr, "sites: remove site directory: %v\n", err)
		return 1
	}

	fmt.Fprintln(os.Stderr, "deleting Postgres rows...")
	if err := runSiteDeleteSQL(deployDir, removed.SiteID); err != nil {
		fmt.Fprintf(os.Stderr, "sites: database cleanup: %v\n", err)
		return 1
	}

	fmt.Fprintln(os.Stderr, "reconciling appliance stack...")
	if err := runDockerCompose(deployDir, "up", "-d", "--remove-orphans"); err != nil {
		fmt.Fprintf(os.Stderr, "sites: compose up: %v\n", err)
		return 1
	}
	if err := runSyncSiteTopology(deployDir); err != nil {
		fmt.Fprintf(os.Stderr, "sites: topology sync: %v\n", err)
		return 1
	}

	fmt.Fprintf(os.Stdout, "Deleted site %s (%d site(s) remaining).\n", removed.SiteID, len(updated.Sites))
	return 0
}

func parseSitesDeleteArgs(args []string) (siteID string, yes bool, err error) {
	for _, arg := range args {
		switch arg {
		case "--yes":
			yes = true
		default:
			if strings.HasPrefix(arg, "-") {
				return "", false, fmt.Errorf("sites delete: unknown flag %q", arg)
			}
			if siteID != "" {
				return "", false, fmt.Errorf("usage: equate sites delete <site-id> [--yes]")
			}
			siteID = arg
		}
	}
	if siteID == "" {
		return "", false, fmt.Errorf("usage: equate sites delete <site-id> [--yes]")
	}
	return siteID, yes, nil
}

func runSiteDeleteSQL(deployDir, siteID string) error {
	const composeEnv = "/run/equate/rendered/compose.env"
	sql := setup.SiteDeleteSQL(siteID)
	cmd := exec.Command("docker", append(dockerComposeBaseArgs(deployDir), "exec", "-T", "postgres",
		"psql", "-U", postgresUserFromEnv(composeEnv), "-d", postgresDBFromEnv(composeEnv),
		"-v", "ON_ERROR_STOP=1")...)
	cmd.Dir = deployDir
	cmd.Stdin = strings.NewReader(sql + "\n")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func dockerComposeBaseArgs(deployDir string) []string {
	args := []string{"compose"}
	const composeEnv = "/run/equate/rendered/compose.env"
	if _, err := os.Stat(composeEnv); err == nil {
		args = append(args, "--env-file", composeEnv)
	}
	for _, f := range dockerComposeFiles(deployDir) {
		if _, err := os.Stat(f); err == nil {
			args = append(args, "-f", f)
		}
	}
	return args
}

func postgresUserFromEnv(composeEnv string) string {
	if v := lookupEnvFile(composeEnv, "POSTGRES_USER"); v != "" {
		return v
	}
	return "postgres"
}

func postgresDBFromEnv(composeEnv string) string {
	if v := lookupEnvFile(composeEnv, "POSTGRES_DB"); v != "" {
		return v
	}
	return "ogsd"
}

func lookupEnvFile(path, key string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	prefix := key + "="
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, prefix)), `"'`)
		}
	}
	return ""
}

func runSyncSiteTopology(deployDir string) error {
	script := filepath.Join(deployDir, "scripts", "sync-site-topology.sh")
	if _, err := os.Stat(script); err != nil {
		return nil
	}
	cmd := exec.Command("bash", script)
	cmd.Dir = deployDir
	cmd.Env = append(os.Environ(),
		"EQUATE_DEPLOY_DIR="+deployDir,
		"EQUATE_COMPOSE_ENV=/run/equate/rendered/compose.env",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
