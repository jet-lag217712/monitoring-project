package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"github.com/equate/ogsd/services/snmp-collector/internal/discovery"
	"github.com/equate/ogsd/services/snmp-collector/internal/metrics"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/vendors"
)

func runDiscover(args []string) int {
	if len(args) > 0 {
		switch args[0] {
		case "export":
			return runDiscoverExport(args[1:])
		case "accept":
			return runDiscoverAccept(args[1:])
		}
	}

	fs := flag.NewFlagSet("discover", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "configs/collector.example.yaml", "path to collector config file")
	outputPath := fs.String("output", "discovery-candidates.json", "path for the reviewable candidate file")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "discover accepts no positional arguments; use discover export|accept for review workflows")
		return 2
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		return 1
	}
	if len(cfg.Discovery.AllowedCIDRs) == 0 {
		fmt.Fprintln(os.Stderr, "discovery is not configured; set discovery.allowed_cidrs and discovery.community_env")
		return 1
	}
	community := strings.TrimSpace(cfg.DiscoveryCommunity())
	if community == "" {
		fmt.Fprintf(os.Stderr, "discovery community environment variable %q is not set\n", cfg.Discovery.CommunityEnv)
		return 1
	}

	m := metrics.New()
	registry := vendors.NewRegistry()
	scanner, err := discovery.New(cfg.Discovery, community, discovery.NewSNMPProber(), discovery.WithProfileDetector(func(identity discovery.Identity) string {
		matched, _ := registry.Match(identity.SysObjectID)
		if matched == nil {
			return "core"
		}
		return matched.Name()
	}), discovery.WithRateLimitWaitObserver(func() {
		m.DiscoveryRateLimitWaits.Inc()
	}))
	if err != nil {
		fmt.Fprintf(os.Stderr, "create discovery scanner: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	log.Info("discovery scan starting",
		"allowed_cidrs", cfg.Discovery.AllowedCIDRs,
		"max_targets", cfg.Discovery.MaxTargets,
		"max_workers", cfg.Discovery.MaxWorkers,
		"max_probes_per_second", cfg.Discovery.MaxProbesPerSecond,
		"probe_burst", cfg.Discovery.ProbeBurst,
	)

	started := time.Now()
	candidates, err := scanner.Scan(ctx)
	for _, candidate := range candidates {
		m.DiscoveryAttemptsTotal.Inc()
		if candidate.Result == discovery.ProbeSucceeded {
			m.DiscoveryCandidatesTotal.Inc()
		} else {
			m.DiscoveryErrorsTotal.Inc()
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "discovery scan failed: %v\n", err)
		return 1
	}
	if err := discovery.WriteCandidates(*outputPath, candidates); err != nil {
		fmt.Fprintf(os.Stderr, "write candidates: %v\n", err)
		return 1
	}

	succeeded := 0
	for _, candidate := range candidates {
		if candidate.Result == discovery.ProbeSucceeded {
			succeeded++
		}
	}
	log.Info("discovery scan complete",
		"targets", len(candidates),
		"candidates", succeeded,
		"output", *outputPath,
		"duration", time.Since(started).String(),
	)
	fmt.Fprintf(os.Stdout, "wrote %d probe results (%d candidates) to %s\n", len(candidates), succeeded, *outputPath)
	return 0
}

func runDiscoverExport(args []string) int {
	fs := flag.NewFlagSet("discover export", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fromPath := fs.String("from", "", "path to reviewed candidates file")
	toPath := fs.String("to", "discovery-export.yaml", "path for non-activating inventory export")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*fromPath) == "" {
		fmt.Fprintln(os.Stderr, "-from is required")
		return 2
	}
	reviews, err := discovery.ReadReviews(*fromPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read reviews: %v\n", err)
		return 1
	}
	if err := discovery.ExportReviewed(*toPath, reviews); err != nil {
		fmt.Fprintf(os.Stderr, "export reviewed candidates: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "exported approved candidates to %s\n", *toPath)
	return 0
}

func runDiscoverAccept(args []string) int {
	fs := flag.NewFlagSet("discover accept", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "configs/collector.example.yaml", "path to collector config file")
	fromPath := fs.String("from", "", "path to reviewed candidates file")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*fromPath) == "" {
		fmt.Fprintln(os.Stderr, "-from is required")
		return 2
	}

	cfg, err := config.LoadForValidation(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		return 1
	}
	managedPath := cfg.ManagedInventoryPath()
	if managedPath == "" {
		fmt.Fprintln(os.Stderr, "inventory.managed_path is required to accept discovery candidates")
		return 1
	}
	reviews, err := discovery.ReadReviews(*fromPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read reviews: %v\n", err)
		return 1
	}
	currentManaged, err := config.ReadManagedInventory(managedPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read managed inventory: %v\n", err)
		return 1
	}
	if err := discovery.AcceptReviewed(managedPath, currentManaged, cfg.Devices, reviews, config.WriteManagedInventory); err != nil {
		fmt.Fprintf(os.Stderr, "accept reviewed candidates: %v\n", err)
		return 1
	}

	// Re-validate the full static+managed merge after the write. An invalid
	// result leaves the managed file updated but never schedules polling.
	if _, err := config.LoadForValidation(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "managed inventory was written but full validation failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "accepted approved candidates into %s\n", managedPath)
	fmt.Fprintln(os.Stdout, "reload the running collector (SIGHUP) to activate the new inventory")
	return 0
}
