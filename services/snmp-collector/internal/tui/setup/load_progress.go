package setup

import (
	"context"
	"fmt"
	"time"
)

const discoveryPollInterval = 500 * time.Millisecond

func startAsyncDiscoveryScan(client controlCaller) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := client.Call(ctx, "sc1", "discovery.scan.start", map[string]any{"async": true})
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message)
	}
	return nil
}

func pollDiscoveryScanProgress(client controlCaller) (running bool, probed, total int, scanErr string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.Call(ctx, "sc2", "discovery.scan.progress", nil)
	if err != nil {
		return false, 0, 0, "", err
	}
	if !resp.OK {
		return false, 0, 0, "", fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message)
	}
	running, _ = resp.Result["running"].(bool)
	probed = intFromAny(resp.Result["probed"])
	total = intFromAny(resp.Result["total"])
	if raw, ok := resp.Result["error"].(string); ok {
		scanErr = raw
	}
	return running, probed, total, scanErr, nil
}

func listDiscoveryCandidates(client controlCaller) ([]map[string]any, error) {
	listCtx, listCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer listCancel()
	resp, err := client.Call(listCtx, "sc3", "discovery.candidates.list", nil)
	if err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message)
	}
	raw, _ := resp.Result["candidates"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out, nil
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

type deployBeginMsg struct {
	sites []SiteSpec
	err   error
}

type deployPhaseMsg struct {
	step     int
	total    int
	label    string
	next     int
	finished bool
	body     string
	err      error
}

type discoveryPollTickMsg struct{}

type discoveryProgressMsg struct {
	running bool
	probed  int
	total   int
	scanErr string
	err     error
}

type discoveryScanDoneMsg struct {
	candidates []map[string]any
	err        error
}

type reviewAutoBeginMsg struct {
	sites []SiteSpec
	err   error
}

type reviewAutoAcceptMsg struct {
	line string
	err  error
}

type discoveryScanStartedMsg struct {
	spec   SiteSpec
	manual bool
}

func runDeployPhase(deployDir string, profile Profile, sites []SiteSpec, phase int) deployPhaseMsg {
	total := 3 + len(sites)
	services := serviceNames(sites)
	var label string
	var err error

	switch phase {
	case 0:
		label = "Starting containers"
		err = startCompose(deployDir, profile, services)
	case 1:
		label = "Preparing ownership"
		if e := stopCompose(deployDir, services); e != nil {
			err = e
		} else {
			err = ensureSiteOwnership(deployDir, profile, sites)
		}
	case 2:
		label = "Restarting collectors"
		err = restartCompose(deployDir, services)
	default:
		siteIdx := phase - 3
		if siteIdx >= len(sites) {
			return deployPhaseMsg{
				step:     total,
				total:    total,
				finished: true,
				body:     fmt.Sprintf("Started %d collector container(s).", len(sites)),
			}
		}
		spec := sites[siteIdx]
		label = fmt.Sprintf("Waiting for %s", spec.SiteID)
		client := newDeployControl(deployDir, spec)
		err = waitForCollector(spec.AdminURL(), client, 3*time.Minute)
	}

	if err != nil {
		return deployPhaseMsg{step: phase + 1, total: total, label: label, err: err}
	}
	if phase >= 2+len(sites) {
		return deployPhaseMsg{
			step:     total,
			total:    total,
			label:    label,
			finished: true,
			body:     fmt.Sprintf("Started %d collector container(s).", len(sites)),
		}
	}
	return deployPhaseMsg{step: phase + 1, total: total, label: label, next: phase + 1}
}

func acceptSiteDiscovery(spec SiteSpec, deployDir string, candidates []map[string]any) (string, error) {
	if len(candidates) == 0 {
		return fmt.Sprintf("%s: discovery completed with no candidates", spec.SiteID), nil
	}
	success := 0
	for _, c := range candidates {
		if fmt.Sprint(c["result"]) == "success" {
			success++
		}
	}
	if success == 0 {
		return fmt.Sprintf("%s: discovery finished with no successful probes", spec.SiteID), nil
	}
	client := newDeployControl(deployDir, spec)
	if err := acceptAllSuccessful(spec.ManagedInventoryPath(deployDir), "SNMP_COMMUNITY", client, candidates); err != nil {
		return "", fmt.Errorf("%s: %w", spec.SiteID, err)
	}
	return fmt.Sprintf("%s: accepted %d discovered device(s)", spec.SiteID, success), nil
}
