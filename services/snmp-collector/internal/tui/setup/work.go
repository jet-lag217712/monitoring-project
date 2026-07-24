package setup

import (
	"fmt"
	"strings"
	"time"
)

type workKind int

const (
	workNone workKind = iota
	workStart
	workReview
)

type workProgressMsg struct {
	current int
	total   int
	label   string
	next    bool
	done    bool
	err     error
	body    string
	sites   []SiteSpec
	lines   []string
}

func startWorkTotal(sites []SiteSpec) int {
	return 4 + len(sites)
}

func reviewWorkTotal(sites []SiteSpec) int {
	if len(sites) == 0 {
		return 1
	}
	return len(sites)
}

func executeStartStep(deployDir string, step int, sites []SiteSpec) workProgressMsg {
	total := startWorkTotal(sites)
	fail := func(err error) workProgressMsg {
		return workProgressMsg{done: true, err: err}
	}
	progress := func(cur int, label string, next bool) workProgressMsg {
		return workProgressMsg{current: cur, total: total, label: label, next: next, sites: sites}
	}

	switch step {
	case 0:
		manifest, err := LoadManifest(deployDir)
		if err != nil {
			return fail(err)
		}
		sites = manifest.Sites
		total = startWorkTotal(sites)
		services := serviceNames(sites)
		if err := startCompose(deployDir, services); err != nil {
			return fail(err)
		}
		return workProgressMsg{current: 1, total: total, label: "Building collector images…", next: true, sites: sites}
	case 1:
		if err := stopCompose(deployDir, serviceNames(sites)); err != nil {
			return fail(err)
		}
		return progress(2, "Preparing site containers…", true)
	case 2:
		if err := ensureSiteOwnership(deployDir, sites); err != nil {
			return fail(err)
		}
		return progress(3, "Fixing volume permissions…", true)
	case 3:
		if err := restartCompose(deployDir, serviceNames(sites)); err != nil {
			return fail(err)
		}
		return progress(4, "Starting collectors…", true)
	default:
		idx := step - 4
		if idx >= len(sites) {
			return workProgressMsg{
				done: true,
				body: fmt.Sprintf("Started %d collector container(s).", len(sites)),
			}
		}
		spec := sites[idx]
		client := newDeployControl(deployDir, spec)
		if err := waitForCollector(spec.AdminURL(), client, 3*time.Minute); err != nil {
			return fail(fmt.Errorf("%s: %w", spec.SiteID, err))
		}
		label := fmt.Sprintf("Waiting for %s (%d/%d)…", spec.SiteID, idx+1, len(sites))
		if idx+1 == len(sites) {
			return workProgressMsg{
				current: total,
				total:   total,
				label:   label,
				done:    true,
				body:    fmt.Sprintf("Started %d collector container(s).", len(sites)),
				sites:   sites,
			}
		}
		return progress(4+idx+1, label, true)
	}
}

func executeReviewStep(deployDir string, step int, sites []SiteSpec, lines []string) (workProgressMsg, []string) {
	fail := func(err error) (workProgressMsg, []string) {
		return workProgressMsg{done: true, err: err}, lines
	}

	if step == 0 && len(sites) == 0 {
		if err := loadEnvFile(envPath(deployDir)); err != nil {
			return fail(fmt.Errorf("load .env: %w", err))
		}
		manifest, err := LoadManifest(deployDir)
		if err != nil {
			return fail(err)
		}
		sites = manifest.Sites
	}

	total := reviewWorkTotal(sites)
	if len(sites) == 0 {
		return workProgressMsg{done: true, body: "No sites configured."}, lines
	}

	if step >= len(sites) {
		return workProgressMsg{done: true, body: strings.Join(lines, "\n")}, lines
	}

	spec := sites[step]
	line, err := reviewSite(spec, deployDir)
	if err != nil {
		return fail(err)
	}
	lines = append(lines, line)
	label := fmt.Sprintf("Discovering %s (%d/%d)…", spec.SiteID, step+1, len(sites))
	if step+1 == len(sites) {
		return workProgressMsg{
			current: step + 1,
			total:   total,
			label:   label,
			done:    true,
			body:    strings.Join(lines, "\n"),
			sites:   sites,
			lines:   lines,
		}, lines
	}
	return workProgressMsg{
		current: step + 1,
		total:   total,
		label:   label,
		next:    true,
		sites:   sites,
		lines:   lines,
	}, lines
}
