package appliance

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"filippo.io/age"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"gopkg.in/yaml.v3"
)

type consoleScreen int

const (
	screenHome consoleScreen = iota
	screenSetup
)

type setupField struct {
	label       string
	defaultText string
	secret      bool
}

var snmpSetupFields = []setupField{
	{label: "Google Workspace domains (comma separated)"},
	{label: "Default SNMP community", secret: true},
	{label: "Discovery SNMP community (blank uses default)", secret: true},
	{label: "Discovery CIDRs (comma separated)"},
	{label: "Discovery probes per second", defaultText: "5"},
	{label: "Discovery burst", defaultText: "5"},
	{label: "Temperature warning °C", defaultText: "65"},
}

type consoleModel struct {
	layout       Layout
	screen       consoleScreen
	setupIndex   int
	setupValues  []string
	setupMessage string
	tlsHostname  string
}

// RunConsole starts the only appliance configuration experience: the local
// SNMP setup and operator TUI. Host/network/auth menus are intentionally absent.
func RunConsole(layout Layout) error {
	screen := screenHome
	if _, err := os.Stat(layout.SetupMarker); os.IsNotExist(err) {
		return RunSNMPSetup(layout)
	}
	p := tea.NewProgram(consoleModel{layout: layout, screen: screen, setupIndex: -1, setupValues: make([]string, len(snmpSetupFields))}, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// RunSNMPSetup always opens the first-run/reconfiguration flow.
func RunSNMPSetup(layout Layout) error {
	p := tea.NewProgram(consoleModel{layout: layout, screen: screenSetup, setupIndex: -1, setupValues: make([]string, len(snmpSetupFields))}, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// RunSNMPTUI opens the day-two collector TUI against the local Unix socket.
func RunSNMPTUI(layout Layout) error {
	command := exec.Command("/usr/local/bin/collector", "tui", "-socket", filepath.Join(layout.Sockets, "poller.sock"), "-theme", "auto")
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("run SNMP operator TUI: %w", err)
	}
	return nil
}

func (m consoleModel) Init() tea.Cmd { return nil }

func (m consoleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if m.screen == screenSetup {
		return m.updateSetup(key)
	}
	switch key.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "s":
		if err := RunSNMPTUI(m.layout); err != nil {
			m.setupMessage = "SNMP TUI unavailable: " + err.Error()
		}
	case "r":
		m.screen = screenSetup
		m.setupIndex = -1
		m.setupValues = make([]string, len(snmpSetupFields))
		m.setupMessage = ""
		m.tlsHostname = ""
	}
	return m, nil
}

func (m consoleModel) updateSetup(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if m.setupIndex == -1 {
		if key.String() == "enter" {
			hostname, err := m.importTLS()
			if err != nil {
				m.setupMessage = "TLS import failed: " + err.Error()
				return m, nil
			}
			m.tlsHostname = hostname
			m.setupIndex = 0
			m.setupMessage = "TLS certificate imported for " + hostname
		}
		return m, nil
	}
	if m.setupIndex >= len(snmpSetupFields) {
		if key.String() == "enter" {
			if err := m.completeSNMPSetup(); err != nil {
				m.setupMessage = "Setup failed: " + err.Error()
				return m, nil
			}
			m.screen = screenHome
			m.setupMessage = "SNMP setup complete. Press s for the operator TUI."
		}
		return m, nil
	}
	value := m.setupValues[m.setupIndex]
	switch key.String() {
	case "enter":
		if strings.TrimSpace(value) == "" {
			value = snmpSetupFields[m.setupIndex].defaultText
		}
		if m.setupIndex == 0 || m.setupIndex == 1 || m.setupIndex == 3 {
			if strings.TrimSpace(value) == "" {
				m.setupMessage = snmpSetupFields[m.setupIndex].label + " is required."
				return m, nil
			}
		}
		m.setupValues[m.setupIndex] = value
		m.setupIndex++
		m.setupMessage = ""
	case "backspace":
		if len(value) > 0 {
			m.setupValues[m.setupIndex] = value[:len(value)-1]
		}
	case "esc":
		if m.setupIndex > 0 {
			m.setupIndex--
		}
	default:
		if len(key.String()) == 1 && key.String()[0] >= 32 {
			m.setupValues[m.setupIndex] += key.String()
		}
	}
	return m, nil
}

func (m consoleModel) importTLS() (string, error) {
	identity, err := m.secretIdentity()
	if err != nil {
		return "", err
	}
	return m.layout.ImportTLSMedia(identity)
}

func (m consoleModel) completeSNMPSetup() error {
	if m.tlsHostname == "" {
		return fmt.Errorf("import a client CA TLS certificate before continuing")
	}
	identity, err := m.secretIdentity()
	if err != nil {
		return err
	}
	values := func(index int) string { return strings.TrimSpace(m.setupValues[index]) }
	domains, err := parseWorkspaceDomains(values(0))
	if err != nil {
		return err
	}
	community := values(1)
	if strings.ContainsAny(community, "\r\n") {
		return fmt.Errorf("SNMP community cannot contain line breaks")
	}
	discoveryCommunity := values(2)
	if discoveryCommunity == "" {
		discoveryCommunity = community
	}
	cidrs, err := parseCIDRs(values(3))
	if err != nil {
		return err
	}
	rate, err := strconv.ParseFloat(values(4), 64)
	if err != nil || rate <= 0 {
		return fmt.Errorf("discovery probes per second must be positive")
	}
	burst, err := strconv.Atoi(values(5))
	if err != nil || burst <= 0 {
		return fmt.Errorf("discovery burst must be a positive integer")
	}
	threshold, err := strconv.ParseFloat(values(6), 64)
	if err != nil || threshold < -100 || threshold > 250 {
		return fmt.Errorf("temperature warning must be between -100 and 250")
	}
	if err := m.layout.WriteSecret(identity, "snmp.community", []byte(community)); err != nil {
		return fmt.Errorf("encrypt SNMP community: %w", err)
	}
	if err := m.layout.WriteSecret(identity, "snmp.discovery-community", []byte(discoveryCommunity)); err != nil {
		return fmt.Errorf("encrypt discovery community: %w", err)
	}
	if err := updateApplicationDomains(m.layout.ApplicationYML, domains); err != nil {
		return err
	}
	if err := writeInitialManagedInventory(filepath.Join(m.layout.Data, "core", "managed-inventory.yaml"), cidrs, rate, burst, threshold); err != nil {
		return err
	}
	if err := AtomicWriteFile(m.layout.SetupMarker, []byte("complete\n"), 0o600); err != nil {
		return err
	}
	if err := restartConfiguredServices(m.layout); err != nil {
		return err
	}
	return nil
}

func (m consoleModel) secretIdentity() (*age.X25519Identity, error) {
	raw, err := os.ReadFile(m.layout.Identity)
	if err != nil {
		return nil, fmt.Errorf("read appliance secret identity: %w", err)
	}
	return ParseSecretIdentity(raw)
}

func updateApplicationDomains(path string, domains []string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read application configuration: %w", err)
	}
	var app map[string]any
	if err := yaml.Unmarshal(data, &app); err != nil {
		return fmt.Errorf("parse application configuration: %w", err)
	}
	authConfig, _ := app["auth"].(map[string]any)
	if authConfig == nil {
		return fmt.Errorf("application configuration has no auth section")
	}
	clientID, _ := authConfig["google_client_id"].(string)
	if strings.TrimSpace(clientID) == "" || strings.Contains(clientID, "__EQUATE_") {
		return fmt.Errorf("release is missing its Equate-managed Google client ID")
	}
	authConfig["enabled"] = true
	authConfig["mode"] = "google_session"
	authConfig["allowed_domains"] = domains
	app["auth"] = authConfig
	encoded, err := yaml.Marshal(app)
	if err != nil {
		return fmt.Errorf("encode application configuration: %w", err)
	}
	return AtomicWriteFile(path, encoded, 0o644)
}

func writeInitialManagedInventory(path string, cidrs []string, rate float64, burst int, threshold float64) error {
	return config.WriteManagedDocument(path, config.ManagedInventory{
		Health: config.ManagedHealthPolicy{TemperatureWarningC: &threshold},
		Discovery: config.ManagedDiscoveryPolicy{
			AllowedCIDRs:       cidrs,
			CommunityEnv:       "SNMP_DISCOVERY_COMMUNITY",
			MaxProbesPerSecond: &rate,
			ProbeBurst:         &burst,
		},
	})
}

func parseWorkspaceDomains(raw string) ([]string, error) {
	seen := map[string]struct{}{}
	var domains []string
	for _, item := range strings.Split(raw, ",") {
		domain := strings.ToLower(strings.TrimSpace(item))
		if domain == "" {
			continue
		}
		if !validWorkspaceDomain(domain) {
			return nil, fmt.Errorf("invalid Workspace domain %q", domain)
		}
		if _, exists := seen[domain]; !exists {
			seen[domain] = struct{}{}
			domains = append(domains, domain)
		}
	}
	if len(domains) == 0 {
		return nil, fmt.Errorf("at least one Google Workspace domain is required")
	}
	return domains, nil
}

func validWorkspaceDomain(domain string) bool {
	if len(domain) > 253 || !strings.Contains(domain, ".") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}
	for _, label := range strings.Split(domain, ".") {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		for _, char := range label {
			if !(char == '-' || char >= 'a' && char <= 'z' || char >= '0' && char <= '9') {
				return false
			}
		}
	}
	return true
}

func parseCIDRs(raw string) ([]string, error) {
	seen := map[string]struct{}{}
	var cidrs []string
	for _, item := range strings.Split(raw, ",") {
		cidr := strings.TrimSpace(item)
		if cidr == "" {
			continue
		}
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return nil, fmt.Errorf("invalid discovery CIDR %q", cidr)
		}
		if _, exists := seen[cidr]; !exists {
			seen[cidr] = struct{}{}
			cidrs = append(cidrs, cidr)
		}
	}
	if len(cidrs) == 0 {
		return nil, fmt.Errorf("at least one discovery CIDR is required")
	}
	return cidrs, nil
}

func restartConfiguredServices(layout Layout) error {
	if layout.Root != "/" {
		return nil
	}
	if output, err := exec.Command("systemctl", "restart", "equate-init.service").CombinedOutput(); err != nil {
		return fmt.Errorf("render appliance secrets: %w: %s", err, strings.TrimSpace(string(output)))
	}
	current, err := layout.CurrentRelease()
	if err != nil {
		return err
	}
	command := exec.Command("docker", "compose", "--project-name", "equate", "--env-file", filepath.Join(layout.Rendered, "compose.env"), "--file", filepath.Join(current, "compose.yaml"), "up", "--detach", "--no-deps", "--force-recreate", "api", "equate-core", "ui")
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("restart configured services: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (m consoleModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render("Equate Appliance")
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	if m.screen == screenSetup {
		return strings.Join([]string{title, "", m.setupView(), "", muted.Render("Local SNMP setup only · credentials are encrypted · no Linux shell")}, "\n") + "\n"
	}
	version := "unconfigured"
	if current, err := m.layout.CurrentRelease(); err == nil {
		version = "v" + filepath.Base(current)
	}
	body := fmt.Sprintf("System Status\n\nRelease: %s\nDashboard: starts automatically on HTTPS\n\ns  SNMP operator TUI\nr  Reconfigure local SNMP setup\nq  Exit", version)
	if m.setupMessage != "" {
		body += "\n\n" + m.setupMessage
	}
	return strings.Join([]string{title, "", body, "", muted.Render("Local console only · no Linux shell")}, "\n") + "\n"
}

func (m consoleModel) setupView() string {
	if m.setupIndex == -1 {
		view := "SNMP Setup\n\nAttach client-CA certificate media labeled EQUATE_TLS containing tls.crt and tls.key.\n\nPress Enter to import and validate the certificate."
		if m.setupMessage != "" {
			view += "\n\n" + m.setupMessage
		}
		return view
	}
	if m.setupIndex >= len(snmpSetupFields) {
		return "SNMP Setup\n\nReview complete. Press Enter to encrypt credentials, save the managed discovery policy, and activate Google Workspace sign-in.\n\n" + m.setupMessage
	}
	field := snmpSetupFields[m.setupIndex]
	value := m.setupValues[m.setupIndex]
	if field.secret && value != "" {
		value = strings.Repeat("•", len(value))
	}
	if value == "" && !field.secret {
		value = field.defaultText
	}
	view := fmt.Sprintf("SNMP Setup (%d/%d)\n\n%s\n> %s\n\nEnter  Continue", m.setupIndex+1, len(snmpSetupFields), field.label, value)
	if m.setupMessage != "" {
		view += "\n\n" + m.setupMessage
	}
	return view
}
