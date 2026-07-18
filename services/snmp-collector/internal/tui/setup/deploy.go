package setup

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"github.com/equate/ogsd/services/snmp-collector/internal/topology"
)

const setupMarker = ".setup-complete"

func writeEnvFile(path string, values map[string]string) error {
	var b strings.Builder
	keys := []string{"MQTT_BROKER", "MQTT_PASSWORD", "SNMP_COMMUNITY", "SNMP_DISCOVERY_COMMUNITY"}
	seen := make(map[string]struct{})
	for _, k := range keys {
		if v, ok := values[k]; ok && strings.TrimSpace(v) != "" {
			fmt.Fprintf(&b, "%s=%s\n", k, v)
			seen[k] = struct{}{}
		}
	}
	for k, v := range values {
		if _, ok := seen[k]; ok {
			continue
		}
		if strings.TrimSpace(v) != "" {
			fmt.Fprintf(&b, "%s=%s\n", k, v)
		}
	}
	return os.WriteFile(path, []byte(b.String()), 0o600)
}

func applyEnvToProcess(values map[string]string) {
	for k, v := range values {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		_ = os.Setenv(k, v)
	}
}

func loadEnvFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key == "" {
			continue
		}
		if err := os.Setenv(key, val); err != nil {
			return fmt.Errorf("set %s: %w", key, err)
		}
	}
	return nil
}

func writeSeedManaged(path string, cidrs []string, rate float64, burst int, communityEnv string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	rateCopy := rate
	burstCopy := burst
	doc := config.ManagedInventory{
		Discovery: config.ManagedDiscoveryPolicy{
			AllowedCIDRs:       cidrs,
			CommunityEnv:       communityEnv,
			MaxProbesPerSecond: &rateCopy,
			ProbeBurst:         &burstCopy,
		},
		Devices: []config.DeviceConfig{},
	}
	data, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func startCompose(deployDir string, services []string) error {
	args := []string{"compose", "-f", "docker-compose.yml", "-f", generatedComposeFile, "up", "-d", "--build", "--remove-orphans"}
	args = append(args, services...)
	// #region agent log
	agentLog("A", "deploy.go:startCompose", "starting docker compose", "pre-fix", map[string]any{
		"deployDir": deployDir,
		"services":  services,
		"args":      args,
	})
	// #endregion
	cmd := exec.Command("docker", args...)
	cmd.Dir = deployDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	// #region agent log
	agentLog("C", "deploy.go:startCompose", "docker compose finished", "pre-fix", map[string]any{
		"exitErr":  errString(err),
		"stderr":   strings.TrimSpace(stderr.String()),
		"stdout":   strings.TrimSpace(stdout.String()),
		"services": services,
	})
	// #endregion
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("docker compose: %s", msg)
	}
	return nil
}

func stopCompose(deployDir string, services []string) error {
	args := []string{"compose", "-f", "docker-compose.yml", "-f", generatedComposeFile, "stop"}
	args = append(args, services...)
	cmd := exec.Command("docker", args...)
	cmd.Dir = deployDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("docker compose stop: %s", msg)
	}
	return nil
}

func restartCompose(deployDir string, services []string) error {
	args := []string{"compose", "-f", "docker-compose.yml", "-f", generatedComposeFile, "up", "-d", "--remove-orphans"}
	args = append(args, services...)
	cmd := exec.Command("docker", args...)
	cmd.Dir = deployDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("docker compose restart: %s", msg)
	}
	return nil
}

const composeProjectName = "ogsd-development-vxrail"

func ensureSiteOwnership(deployDir string, specs []SiteSpec) error {
	for _, spec := range specs {
		volume := composeProjectName + "_" + spec.VolumeName()
		// #region agent log
		agentLog("F", "deploy.go:ensureSiteOwnership", "chown site paths", "post-fix", map[string]any{
			"siteID": spec.SiteID,
			"volume": volume,
		})
		// #endregion
		if err := chownRunDir(spec.RunDir(deployDir)); err != nil {
			// #region agent log
			agentLog("H", "deploy.go:ensureSiteOwnership", "run dir chown failed", "post-fix", map[string]any{
				"siteID": spec.SiteID,
				"error":  err.Error(),
			})
			// #endregion
			return fmt.Errorf("%s run dir: %w", spec.SiteID, err)
		}
		// #region agent log
		agentLog("H", "deploy.go:ensureSiteOwnership", "run dir chown ok", "post-fix", map[string]any{
			"siteID": spec.SiteID,
		})
		// #endregion
		if err := chownContainerPath(volume+":/var/lib/snmp-collector", "/var/lib/snmp-collector", true); err != nil {
			return fmt.Errorf("%s state volume: %w", spec.SiteID, err)
		}
		if err := chownContainerPath(spec.ManagedDir(deployDir)+":/var/lib/snmp-collector/managed", "/var/lib/snmp-collector/managed", true); err != nil {
			return fmt.Errorf("%s managed dir: %w", spec.SiteID, err)
		}
	}
	return nil
}

func chownContainerPath(mount, path string, recursive bool) error {
	args := []string{"run", "--rm", "-v", mount, "busybox:1.36", "chown"}
	if recursive {
		args = append(args, "-R")
	}
	args = append(args, "65532:65532", path)
	cmd := exec.Command("docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

// chownRunDir fixes run-dir ownership without touching Unix sockets (chown -R fails on
// bind-mounted control.sock with EINVAL on Docker Desktop/macOS).
func chownRunDir(hostRunDir string) error {
	mount := hostRunDir + ":/run/snmp-collector"
	if err := runBusybox(mount, "rm", "-f", "/run/snmp-collector/control.sock"); err != nil {
		return err
	}
	return chownContainerPath(mount, "/run/snmp-collector", false)
}

func runBusybox(mount string, args ...string) error {
	cmdArgs := []string{"run", "--rm", "-v", mount, "busybox:1.36"}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command("docker", cmdArgs...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func startSiteCollectors(deployDir string, specs []SiteSpec) error {
	services := serviceNames(specs)
	if err := startCompose(deployDir, services); err != nil {
		return err
	}
	if err := stopCompose(deployDir, services); err != nil {
		return err
	}
	if err := ensureSiteOwnership(deployDir, specs); err != nil {
		return err
	}
	return restartCompose(deployDir, services)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func waitForCollector(adminURL string, client controlCaller, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := client.Call(ctx, "ready", "status.summary", nil)
		cancel()
		if err == nil && resp.OK {
			return nil
		}
		cmd := exec.Command("curl", "-fsS", adminURL+"/healthz")
		if err := cmd.Run(); err == nil {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("collector did not become ready within %s", timeout)
}

func markComplete(deployDir string) error {
	return os.WriteFile(filepath.Join(deployDir, setupMarker), []byte(time.Now().UTC().Format(time.RFC3339Nano)+"\n"), 0o600)
}

// deviceIDFromDiscoveryCandidate prefers the SNMP hostname label so accepted
// devices align with existing inventory rows (e.g. DO-CORE.lab -> do-core).
func deviceIDFromDiscoveryCandidate(hostname, ip string) string {
	hostname = strings.TrimSpace(strings.TrimSuffix(hostname, "."))
	if hostname != "" {
		if i := strings.Index(hostname, "."); i > 0 {
			hostname = hostname[:i]
		}
		if id := strings.ToLower(hostname); id != "" {
			return id
		}
	}
	return "discovered-" + strings.ReplaceAll(ip, ".", "-")
}

func envPath(deployDir string) string {
	return filepath.Join(deployDir, ".env")
}

func persistMultiSiteArtifacts(deployDir string, specs []SiteSpec, rate float64, burst int) error {
	manifest := Manifest{
		SiteCount:     len(specs),
		BaseAdminPort: baseAdminPort,
		ProbeRate:     rate,
		ProbeBurst:    burst,
		Sites:         specs,
	}
	if err := WriteManifest(deployDir, manifest); err != nil {
		return err
	}
	if err := WriteSiteArtifacts(deployDir, specs, rate, burst, "SNMP_DISCOVERY_COMMUNITY"); err != nil {
		return err
	}
	buildContext := "../../../services/snmp-collector"
	if _, err := os.Stat(filepath.Join(deployDir, "src", "services", "snmp-collector", "go.mod")); err == nil {
		buildContext = "./src/services/snmp-collector"
	}
	return GenerateCompose(deployDir, specs, buildContext)
}

func acceptAllSuccessful(managedInventoryPath, communityEnv string, client controlCaller, candidates []map[string]any) error {
	reviews := make([]map[string]any, 0)
	for _, c := range candidates {
		if fmt.Sprint(c["result"]) != "success" {
			continue
		}
		ip := fmt.Sprint(c["ip"])
		id := deviceIDFromDiscoveryCandidate(fmt.Sprint(c["hostname"]), ip)
		reviews = append(reviews, map[string]any{
			"approved": true,
			"candidate": map[string]any{
				"ip":               ip,
				"fingerprint":      c["fingerprint"],
				"detected_profile": c["detected_profile"],
				"hostname":         c["hostname"],
				"description":      c["description"],
				"result":           "success",
			},
			"device": map[string]any{
				"id":            id,
				"host":          ip,
				"port":          161,
				"community_env": communityEnv,
				"version":       "2c",
			},
		})
	}
	if len(reviews) == 0 {
		return fmt.Errorf("no successful candidates to accept")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	prepare, err := client.Call(ctx, "ap1", "discovery.accept.prepare", map[string]any{"reviews": reviews})
	if err != nil {
		return err
	}
	if !prepare.OK {
		return fmt.Errorf("%s: %s", prepare.Error.Code, prepare.Error.Message)
	}
	commit, err := client.Call(ctx, "ap2", "discovery.accept.commit", map[string]any{
		"confirm_token": prepare.Result["confirm_token"],
		"revision":      prepare.Result["revision"],
	})
	if err != nil {
		return err
	}
	if !commit.OK {
		return fmt.Errorf("%s: %s", commit.Error.Code, commit.Error.Message)
	}
	if err := enrichManagedTopology(managedInventoryPath, communityEnv); err != nil {
		return err
	}
	reload, err := client.Call(ctx, "ap3", "config.reload", nil)
	if err != nil {
		return err
	}
	if !reload.OK {
		return fmt.Errorf("%s: %s", reload.Error.Code, reload.Error.Message)
	}
	return nil
}

func enrichManagedTopology(managedInventoryPath, communityEnv string) error {
	doc, err := config.ReadManagedDocument(managedInventoryPath)
	if err != nil {
		return err
	}
	if len(doc.Devices) == 0 {
		return nil
	}
	community := os.Getenv(communityEnv)
	// #region agent log
	agentLog("I", "deploy.go:enrichManagedTopology", "community env lookup", "post-fix", map[string]any{
		"communityEnv": communityEnv,
		"isSet":        strings.TrimSpace(community) != "",
	})
	// #endregion
	if strings.TrimSpace(community) == "" {
		return fmt.Errorf("environment variable %q is not set", communityEnv)
	}
	doc.Devices = topology.Enrich(doc.Devices, community, nil)
	return config.WriteManagedDocument(managedInventoryPath, doc)
}

func setThreshold(client controlCaller, temp float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	prepare, err := client.Call(ctx, "th1", "thresholds.prepare", map[string]any{"temperature_warning_c": temp})
	if err != nil {
		return err
	}
	if !prepare.OK {
		return fmt.Errorf("%s: %s", prepare.Error.Code, prepare.Error.Message)
	}
	commit, err := client.Call(ctx, "th2", "thresholds.commit", map[string]any{
		"confirm_token": prepare.Result["confirm_token"],
		"revision":      prepare.Result["revision"],
	})
	if err != nil {
		return err
	}
	if !commit.OK {
		return fmt.Errorf("%s: %s", commit.Error.Code, commit.Error.Message)
	}
	reload, err := client.Call(ctx, "th3", "config.reload", nil)
	if err != nil {
		return err
	}
	if !reload.OK {
		return fmt.Errorf("%s: %s", reload.Error.Code, reload.Error.Message)
	}
	return nil
}

func runDiscoveryScan(client controlCaller) ([]map[string]any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if _, err := client.Call(ctx, "sc1", "discovery.scan.start", nil); err != nil {
		return nil, err
	}
	listCtx, listCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer listCancel()
	resp, err := client.Call(listCtx, "sc2", "discovery.candidates.list", nil)
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

func reviewSite(spec SiteSpec, deployDir string) (string, error) {
	client := newDeployControl(deployDir, spec)
	candidates, err := runDiscoveryScan(client)
	if err != nil {
		return "", err
	}
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
	if err := acceptAllSuccessful(spec.ManagedInventoryPath(deployDir), "SNMP_COMMUNITY", client, candidates); err != nil {
		return "", fmt.Errorf("%s: %w", spec.SiteID, err)
	}
	return fmt.Sprintf("%s: accepted %d discovered device(s)", spec.SiteID, success), nil
}

func applyThresholdToSite(spec SiteSpec, deployDir string, temp float64) error {
	client := newDeployControl(deployDir, spec)
	if err := setThreshold(client, temp); err != nil {
		return fmt.Errorf("%s: %w", spec.SiteID, err)
	}
	return nil
}

func waitForSites(deployDir string, specs []SiteSpec, timeout time.Duration) error {
	for _, spec := range specs {
		// #region agent log
		agentLog("G", "deploy.go:waitForSites", "waiting for site collector", "post-fix", map[string]any{
			"siteID":   spec.SiteID,
			"adminURL": spec.AdminURL(),
		})
		// #endregion
		client := newDeployControl(deployDir, spec)
		if err := waitForCollector(spec.AdminURL(), client, timeout); err != nil {
			// #region agent log
			agentLog("G", "deploy.go:waitForSites", "site collector not ready", "post-fix", map[string]any{
				"siteID": spec.SiteID,
				"error":  err.Error(),
			})
			// #endregion
			return fmt.Errorf("%s: %w", spec.SiteID, err)
		}
	}
	return nil
}
