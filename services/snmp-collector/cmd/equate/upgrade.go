package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/update"
)

func resolveConfigureScript(deployDir, bundle string) (string, error) {
	candidates := []string{}
	if strings.TrimSpace(bundle) != "" {
		candidates = append(candidates, filepath.Join(bundle, "scripts", "configure-vm.sh"))
	}
	candidates = append(candidates,
		filepath.Join(deployDir, "scripts", "configure-vm.sh"),
		"/tmp/equate-staging/configure-vm.sh",
	)
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("configure-vm.sh not found (checked bundle, release scripts, and /tmp/equate-staging)")
}

func runUpgrade(args []string) int {
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	bundle := fs.String("bundle", "", "path to staged release bundle directory")
	version := fs.String("version", "", "target release version (semver)")
	canary := fs.Bool("canary", false, "roll out collectors one site at a time with health checks")
	rollback := fs.Bool("rollback", false, "roll back to the previous release")
	yes := fs.Bool("yes", false, "skip interactive confirmation")
	check := fs.Bool("check", false, "report current vs channel latest without downloading")
	directURL := fs.String("url", "", "download a specific .eqa URL (requires --sha256 and --signature)")
	sha256Flag := fs.String("sha256", "", "expected SHA-256 for --url downloads")
	sigFlag := fs.String("signature", "", "base64 Ed25519 signature for --url downloads")
	channelConfig := fs.String("channel-config", update.DefaultChannelConfigPath, "path to update-channel.conf")
	allowHTTP := fs.Bool("allow-insecure-http", false, "allow http:// channel/artifact URLs (local testing only)")
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

	if *rollback {
		return runUpgradeRollback(deployDir, *yes)
	}

	hasBundle := strings.TrimSpace(*bundle) != ""
	hasVersion := strings.TrimSpace(*version) != ""
	if hasBundle || hasVersion {
		if !hasBundle || !hasVersion {
			fmt.Fprintln(os.Stderr, "upgrade requires both --bundle and --version for offline mode")
			return 2
		}
		return runUpgradeApply(deployDir, *bundle, *version, *canary, *yes)
	}

	return runConnectedUpgrade(deployDir, connectedUpgradeOpts{
		checkOnly:     *check,
		directURL:     strings.TrimSpace(*directURL),
		directSHA256:  strings.TrimSpace(*sha256Flag),
		directSig:     strings.TrimSpace(*sigFlag),
		channelConfig: *channelConfig,
		allowHTTP:     *allowHTTP,
		canary:        *canary,
		yes:           *yes,
	})
}

type connectedUpgradeOpts struct {
	checkOnly     bool
	directURL     string
	directSHA256  string
	directSig     string
	channelConfig string
	allowHTTP     bool
	canary        bool
	yes           bool
}

func runConnectedUpgrade(deployDir string, opts connectedUpgradeOpts) int {
	current, err := update.CurrentVersion(deployDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "upgrade: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "Installed version: %s\n", current)

	if opts.directURL != "" {
		if opts.checkOnly {
			fmt.Fprintln(os.Stderr, "upgrade: --check cannot be combined with --url")
			return 2
		}
		if opts.directSHA256 == "" || opts.directSig == "" {
			fmt.Fprintln(os.Stderr, "upgrade: --url requires --sha256 and --signature")
			return 2
		}
		return downloadVerifyAndApply(deployDir, current, "unknown", update.ArtifactManifest{
			Artifact:  filepath.Base(opts.directURL),
			URL:       opts.directURL,
			SHA256:    opts.directSHA256,
			Signature: opts.directSig,
		}, opts)
	}

	cfg, err := update.LoadChannelConfig(opts.channelConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "upgrade: %v\n", err)
		return 1
	}

	if cfg == nil {
		if opts.checkOnly {
			fmt.Fprintln(os.Stderr, "upgrade: no update channel configured (missing /etc/equate/update-channel.conf)")
			fmt.Fprintln(os.Stderr, "offline: sudo equate upgrade --bundle /tmp/equate-staging/bundle --version <semver>")
			return 1
		}
		localBundle := "/tmp/equate-staging/bundle"
		if _, err := os.Stat(filepath.Join(localBundle, "release.env")); err == nil {
			ver, err := update.CurrentVersion(localBundle)
			if err != nil {
				fmt.Fprintf(os.Stderr, "upgrade: staged bundle: %v\n", err)
				return 1
			}
			fmt.Fprintf(os.Stderr, "Found staged bundle at %s (version %s)\n", localBundle, ver)
			available, err := update.UpdateAvailable(current, ver)
			if err != nil {
				fmt.Fprintf(os.Stderr, "upgrade: %v\n", err)
				return 1
			}
			if !available {
				fmt.Fprintf(os.Stderr, "Staged version %s is not newer than installed %s\n", ver, current)
				return 0
			}
			return runUpgradeApply(deployDir, localBundle, ver, opts.canary, opts.yes)
		}
		fmt.Fprintln(os.Stderr, "upgrade: no update channel configured and no staged bundle at /tmp/equate-staging/bundle")
		fmt.Fprintln(os.Stderr, "configure /etc/equate/update-channel.conf or stage a bundle, then retry")
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	man, err := update.FetchManifest(ctx, cfg.ChannelURL, opts.allowHTTP)
	if err != nil {
		fmt.Fprintf(os.Stderr, "upgrade: %v\n", err)
		return 1
	}
	latest, art, err := update.SelectArtifact(man, cfg.Edition, update.HostArch())
	if err != nil {
		fmt.Fprintf(os.Stderr, "upgrade: %v\n", err)
		return 1
	}
	if rel, ok := man.Releases[latest]; ok {
		okMin, err := update.MeetsMinVersion(current, rel.MinVersion)
		if err != nil {
			fmt.Fprintf(os.Stderr, "upgrade: %v\n", err)
			return 1
		}
		if !okMin {
			fmt.Fprintf(os.Stderr, "upgrade: installed %s is below minimum %s required for %s\n", current, rel.MinVersion, latest)
			return 1
		}
	}

	fmt.Fprintf(os.Stderr, "Channel %s (%s): latest %s\n", man.Channel, man.Edition, latest)
	available, err := update.UpdateAvailable(current, latest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "upgrade: %v\n", err)
		return 1
	}
	if !available {
		fmt.Fprintf(os.Stderr, "Already up to date (%s)\n", current)
		return 0
	}
	fmt.Fprintf(os.Stderr, "Update available: %s -> %s\n", current, latest)
	if opts.checkOnly {
		return 0
	}
	return downloadVerifyAndApply(deployDir, current, latest, art, opts)
}

func downloadVerifyAndApply(deployDir, current, target string, art update.ArtifactManifest, opts connectedUpgradeOpts) int {
	if art.SHA256 == "" || art.Signature == "" {
		fmt.Fprintln(os.Stderr, "upgrade: refusing download without sha256 and signature")
		return 2
	}
	if !opts.yes {
		fmt.Fprintf(os.Stderr, "Download and upgrade appliance %s -> %s from %s.\n", current, target, update.RedactURL(art.URL))
		fmt.Fprint(os.Stderr, "Type UPGRADE to continue: ")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		if strings.TrimSpace(line) != "UPGRADE" {
			fmt.Fprintln(os.Stderr, "upgrade cancelled")
			return 1
		}
	}

	dest := update.ArtifactDownloadPath(update.DefaultDownloadDir, art.Artifact)
	fmt.Fprintf(os.Stderr, "Downloading to %s ...\n", dest)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()
	err := update.Download(ctx, art.URL, dest, opts.allowHTTP, func(written, total int64) {
		if total > 0 {
			fmt.Fprintf(os.Stderr, "\r  %d / %d bytes (%.0f%%)", written, total, float64(written)*100/float64(total))
		} else {
			fmt.Fprintf(os.Stderr, "\r  %d bytes", written)
		}
	})
	fmt.Fprintln(os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "upgrade: download: %v\n", err)
		return 1
	}

	pub, err := update.PublicKey()
	if err != nil {
		fmt.Fprintf(os.Stderr, "upgrade: public key: %v\n", err)
		return 1
	}
	fmt.Fprintln(os.Stderr, "Verifying SHA-256 and signature...")
	if err := update.Verify(dest, art.SHA256, art.Signature, pub); err != nil {
		fmt.Fprintf(os.Stderr, "upgrade: verify failed: %v\n", err)
		return 1
	}

	staging := update.DefaultStagingDir
	fmt.Fprintf(os.Stderr, "Extracting to %s ...\n", staging)
	if err := update.Extract(dest, staging); err != nil {
		fmt.Fprintf(os.Stderr, "upgrade: extract: %v\n", err)
		return 1
	}
	extracted, err := update.CurrentVersion(staging)
	if err != nil {
		fmt.Fprintf(os.Stderr, "upgrade: %v\n", err)
		return 1
	}
	if target != "unknown" && extracted != target {
		fmt.Fprintf(os.Stderr, "upgrade: extracted version %s does not match channel version %s\n", extracted, target)
		return 1
	}
	return runUpgradeApply(deployDir, staging, extracted, opts.canary, true)
}

func runUpgradeRollback(deployDir string, yes bool) int {
	configureScript, err := resolveConfigureScript(deployDir, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "upgrade: %v\n", err)
		return 1
	}
	if !yes {
		fmt.Fprint(os.Stderr, "This rolls back to the previous appliance release. Type ROLLBACK to continue: ")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		if strings.TrimSpace(line) != "ROLLBACK" {
			fmt.Fprintln(os.Stderr, "upgrade cancelled")
			return 1
		}
	}
	return runConfigureVM(configureScript, "--rollback")
}

func runUpgradeApply(deployDir, bundle, version string, canary, yes bool) int {
	configureScript, err := resolveConfigureScript(deployDir, bundle)
	if err != nil {
		fmt.Fprintf(os.Stderr, "upgrade: %v\n", err)
		return 1
	}
	if !yes {
		fmt.Fprintf(os.Stderr, "Upgrade appliance to version %s from bundle %s.\n", version, bundle)
		fmt.Fprint(os.Stderr, "Type UPGRADE to continue: ")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		if strings.TrimSpace(line) != "UPGRADE" {
			fmt.Fprintln(os.Stderr, "upgrade cancelled")
			return 1
		}
	}
	cmdArgs := []string{"--upgrade", "--bundle", bundle, "--version", version}
	if canary {
		cmdArgs = append(cmdArgs, "--canary")
	}
	return runConfigureVM(configureScript, cmdArgs...)
}

func runConfigureVM(configureScript string, args ...string) int {
	cmdArgs := append([]string{"bash", configureScript}, args...)
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
