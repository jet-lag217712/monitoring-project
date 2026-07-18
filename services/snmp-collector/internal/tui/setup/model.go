package setup

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/equate/ogsd/services/snmp-collector/internal/tui"
)

type step int

const (
	stepEnv step = iota
	stepSites
	stepStart
	stepReview
	stepThresholds
	stepDone
)

type model struct {
	deployDir string
	theme     tui.Theme
	step      step
	err       string
	body      string
	loading   bool
	spinner   spinner.Model

	envInputs      []textinput.Model
	envFocus       int
	siteCountInput textinput.Model
	siteIDInputs   []textinput.Model
	cidrInputs     []textinput.Model
	siteFocus      int
	thresholdInput textinput.Model

	sites         []SiteSpec
	probeRate     string
	probeBurst    string
	reviewResults []string
}

func newModel(deployDir string, theme tui.Theme) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = theme.Spinner

	fields := []struct {
		placeholder string
		mask        bool
	}{
		{"MQTT broker (tls://host:8883)", false},
		{"MQTT password", true},
		{"SNMP community", true},
		{"Discovery community (often same)", true},
	}
	envInputs := make([]textinput.Model, len(fields))
	for i, f := range fields {
		envInputs[i] = textinput.New()
		envInputs[i].Placeholder = f.placeholder
		envInputs[i].CharLimit = 256
		envInputs[i].Width = 50
		if f.mask {
			envInputs[i].EchoMode = textinput.EchoPassword
		}
	}
	envInputs[0].Focus()

	siteCountInput := textinput.New()
	siteCountInput.Placeholder = "number of site containers"
	siteCountInput.CharLimit = 2
	siteCountInput.Width = 8
	siteCountInput.SetValue(strconv.Itoa(defaultSiteCount))

	thresholdInput := textinput.New()
	thresholdInput.Placeholder = "temperature warning °C"
	thresholdInput.CharLimit = 4
	thresholdInput.Width = 8
	thresholdInput.SetValue("65")

	m := model{
		deployDir:      deployDir,
		theme:          theme,
		step:           stepEnv,
		spinner:        sp,
		envInputs:      envInputs,
		siteCountInput: siteCountInput,
		thresholdInput: thresholdInput,
		probeRate:      "5",
		probeBurst:     "2",
	}
	m.resizeSiteInputs(defaultSiteCount)
	return m
}

func (m *model) siteFieldCount() int {
	return 1 + 2*len(m.cidrInputs)
}

func (m *model) resizeSiteInputs(count int) {
	if count < minSiteCount {
		count = minSiteCount
	}
	if count > maxSiteCount {
		count = maxSiteCount
	}
	siteIDs := make([]textinput.Model, count)
	cidrs := make([]textinput.Model, count)
	for i := range siteIDs {
		siteIDs[i] = textinput.New()
		siteIDs[i].Placeholder = siteIDForIndex(i + 1)
		siteIDs[i].CharLimit = maxSiteIDLen
		siteIDs[i].Width = 20
		if i < len(m.siteIDInputs) && strings.TrimSpace(m.siteIDInputs[i].Value()) != "" {
			siteIDs[i].SetValue(m.siteIDInputs[i].Value())
		} else {
			siteIDs[i].SetValue(siteIDForIndex(i + 1))
		}

		cidrs[i] = textinput.New()
		cidrs[i].Placeholder = defaultCIDRForIndex(i + 1)
		cidrs[i].CharLimit = 64
		cidrs[i].Width = 24
		if i < len(m.cidrInputs) && strings.TrimSpace(m.cidrInputs[i].Value()) != "" {
			cidrs[i].SetValue(m.cidrInputs[i].Value())
		} else {
			cidrs[i].SetValue(defaultCIDRForIndex(i + 1))
		}
	}
	m.siteIDInputs = siteIDs
	m.cidrInputs = cidrs
	if m.siteFocus >= m.siteFieldCount() {
		m.siteFocus = 0
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

type asyncDoneMsg struct {
	err  error
	body string
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.loading && m.step == stepReview && m.err != "" {
			switch msg.String() {
			case "r", "R":
				m.err = ""
				m.loading = true
				return m, m.runReview()
			case "s", "S":
				m.err = ""
				m.body = "Skipped discovery; add devices later from the operator TUI."
				m.step = stepThresholds
				return m, nil
			}
		}
		if m.step == stepDone && (msg.String() == "q" || msg.String() == "enter") {
			return m, tea.Quit
		}
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			if m.step == stepDone {
				return m, tea.Quit
			}
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.loading {
			return m, cmd
		}
	case asyncDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			if m.step == stepReview && isDiscoveryRetryable(msg.err) {
				m.err = "discovery timed out or was interrupted"
			}
			return m, nil
		}
		m.err = ""
		m.body = msg.body
		switch m.step {
		case stepSites:
			m.step = stepStart
			return m, nil
		case stepStart:
			m.step = stepReview
			return m, nil
		case stepReview:
			m.step = stepThresholds
			return m, nil
		case stepThresholds:
			m.step = stepDone
			return m, nil
		}
		return m, nil
	}

	if m.loading {
		return m, nil
	}

	switch m.step {
	case stepEnv:
		return m.updateEnv(msg)
	case stepSites:
		return m.updateSites(msg)
	case stepStart:
		return m.updateStart(msg)
	case stepReview:
		if m.err == "" {
			return m.updateReview(msg)
		}
		return m, nil
	case stepThresholds:
		return m.updateThresholds(msg)
	}
	return m, nil
}

func (m model) updateEnv(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "tab", "down":
		m.envFocus = (m.envFocus + 1) % len(m.envInputs)
		return m.updateEnvFocus(), textinput.Blink
	case "shift+tab", "up":
		m.envFocus = (m.envFocus - 1 + len(m.envInputs)) % len(m.envInputs)
		return m.updateEnvFocus(), textinput.Blink
	case "enter":
		if m.envFocus < len(m.envInputs)-1 {
			m.envFocus++
			return m.updateEnvFocus(), textinput.Blink
		}
		m.step = stepSites
		m.siteFocus = 0
		m.siteCountInput.Focus()
		for i := range m.cidrInputs {
			m.cidrInputs[i].Blur()
		}
		for i := range m.siteIDInputs {
			m.siteIDInputs[i].Blur()
		}
		return m, textinput.Blink
	}
	var cmd tea.Cmd
	m.envInputs[m.envFocus], cmd = m.envInputs[m.envFocus].Update(msg)
	return m, cmd
}

func (m model) updateSites(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		if m.siteFocus == 0 {
			prev := m.siteCountInput.Value()
			m.siteCountInput, cmd = m.siteCountInput.Update(msg)
			if count, err := parseSiteCount(m.siteCountInput.Value()); err == nil && m.siteCountInput.Value() != prev {
				m.resizeSiteInputs(count)
			}
			return m, cmd
		}
		if m.siteFocus%2 == 1 {
			idx := (m.siteFocus - 1) / 2
			m.siteIDInputs[idx], cmd = m.siteIDInputs[idx].Update(msg)
		} else {
			idx := m.siteFocus/2 - 1
			m.cidrInputs[idx], cmd = m.cidrInputs[idx].Update(msg)
		}
		return m, cmd
	}
	lastFocus := 2 * len(m.cidrInputs)
	switch key.String() {
	case "tab", "down":
		m.siteFocus = (m.siteFocus + 1) % m.siteFieldCount()
		return m.updateSiteFocus(), textinput.Blink
	case "shift+tab", "up":
		m.siteFocus = (m.siteFocus - 1 + m.siteFieldCount()) % m.siteFieldCount()
		return m.updateSiteFocus(), textinput.Blink
	case "enter":
		if m.siteFocus < lastFocus {
			m.siteFocus++
			return m.updateSiteFocus(), textinput.Blink
		}
		m.loading = true
		m.err = ""
		return m, m.persistEnvAndSites()
	}
	var cmd tea.Cmd
	if m.siteFocus == 0 {
		prev := m.siteCountInput.Value()
		m.siteCountInput, cmd = m.siteCountInput.Update(msg)
		if count, err := parseSiteCount(m.siteCountInput.Value()); err == nil && m.siteCountInput.Value() != prev {
			m.resizeSiteInputs(count)
		}
	} else if m.siteFocus%2 == 1 {
		idx := (m.siteFocus - 1) / 2
		m.siteIDInputs[idx], cmd = m.siteIDInputs[idx].Update(msg)
	} else {
		idx := m.siteFocus/2 - 1
		m.cidrInputs[idx], cmd = m.cidrInputs[idx].Update(msg)
	}
	return m, cmd
}

func (m model) updateStart(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if key.String() == "enter" {
		m.loading = true
		m.err = ""
		return m, m.runStart()
	}
	return m, nil
}

func (m model) updateReview(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if key.String() == "enter" {
		m.loading = true
		m.err = ""
		return m, m.runReview()
	}
	return m, nil
}

func (m model) updateThresholds(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		m.thresholdInput, cmd = m.thresholdInput.Update(msg)
		return m, cmd
	}
	if key.String() == "enter" {
		m.loading = true
		return m, m.applyThreshold()
	}
	var cmd tea.Cmd
	m.thresholdInput, cmd = m.thresholdInput.Update(msg)
	return m, cmd
}

func (m model) updateEnvFocus() model {
	for i := range m.envInputs {
		if i == m.envFocus {
			m.envInputs[i].Focus()
		} else {
			m.envInputs[i].Blur()
		}
	}
	return m
}

func (m model) updateSiteFocus() model {
	m.siteCountInput.Blur()
	for i := range m.siteIDInputs {
		m.siteIDInputs[i].Blur()
		m.cidrInputs[i].Blur()
	}
	if m.siteFocus == 0 {
		m.siteCountInput.Focus()
		return m
	}
	if m.siteFocus%2 == 1 {
		m.siteIDInputs[(m.siteFocus-1)/2].Focus()
	} else {
		m.cidrInputs[m.siteFocus/2-1].Focus()
	}
	return m
}

func (m model) persistEnvAndSites() tea.Cmd {
	deployDir := m.deployDir
	values := map[string]string{
		"MQTT_BROKER":              strings.TrimSpace(m.envInputs[0].Value()),
		"MQTT_PASSWORD":            strings.TrimSpace(m.envInputs[1].Value()),
		"SNMP_COMMUNITY":           strings.TrimSpace(m.envInputs[2].Value()),
		"SNMP_DISCOVERY_COMMUNITY": strings.TrimSpace(m.envInputs[3].Value()),
	}
	if values["SNMP_DISCOVERY_COMMUNITY"] == "" {
		values["SNMP_DISCOVERY_COMMUNITY"] = values["SNMP_COMMUNITY"]
	}
	count, err := parseSiteCount(m.siteCountInput.Value())
	if err != nil {
		return func() tea.Msg { return asyncDoneMsg{err: err} }
	}
	cidrs := make([]string, len(m.cidrInputs))
	for i := range m.cidrInputs {
		cidrs[i] = strings.TrimSpace(m.cidrInputs[i].Value())
	}
	siteIDs := make([]string, len(m.siteIDInputs))
	for i := range m.siteIDInputs {
		siteIDs[i] = strings.TrimSpace(m.siteIDInputs[i].Value())
	}
	rate, _ := strconv.ParseFloat(strings.TrimSpace(m.probeRate), 64)
	if rate <= 0 {
		rate = 5
	}
	burst, _ := strconv.Atoi(strings.TrimSpace(m.probeBurst))
	if burst <= 0 {
		burst = 2
	}
	return func() tea.Msg {
		specs, err := BuildSiteSpecs(count, siteIDs, cidrs)
		if err != nil {
			return asyncDoneMsg{err: err}
		}
		if err := writeEnvFile(envPath(deployDir), values); err != nil {
			return asyncDoneMsg{err: err}
		}
		applyEnvToProcess(values)
		if err := persistMultiSiteArtifacts(deployDir, specs, rate, burst); err != nil {
			return asyncDoneMsg{err: err}
		}
		body := fmt.Sprintf("Saved shared .env, manifest, and %d site artifact(s).", len(specs))
		return asyncDoneMsg{body: body, err: nil}
	}
}

func (m model) runStart() tea.Cmd {
	deployDir := m.deployDir
	return func() tea.Msg {
		manifest, err := LoadManifest(deployDir)
		if err != nil {
			return asyncDoneMsg{err: err}
		}
		if err := startSiteCollectors(deployDir, manifest.Sites); err != nil {
			return asyncDoneMsg{err: err}
		}
		if err := waitForSites(deployDir, manifest.Sites, 3*time.Minute); err != nil {
			return asyncDoneMsg{err: err}
		}
		return asyncDoneMsg{body: fmt.Sprintf("Started %d collector container(s).", len(manifest.Sites))}
	}
}

func (m model) runReview() tea.Cmd {
	deployDir := m.deployDir
	return func() tea.Msg {
		if err := loadEnvFile(envPath(deployDir)); err != nil {
			return asyncDoneMsg{err: fmt.Errorf("load .env: %w", err)}
		}
		manifest, err := LoadManifest(deployDir)
		if err != nil {
			return asyncDoneMsg{err: err}
		}
		lines := make([]string, 0, len(manifest.Sites))
		for _, spec := range manifest.Sites {
			line, err := reviewSite(spec, deployDir)
			if err != nil {
				return asyncDoneMsg{err: err}
			}
			lines = append(lines, line)
		}
		return asyncDoneMsg{body: strings.Join(lines, "\n")}
	}
}

func (m model) applyThreshold() tea.Cmd {
	deployDir := m.deployDir
	temp, _ := strconv.ParseFloat(strings.TrimSpace(m.thresholdInput.Value()), 64)
	if temp <= 0 {
		temp = 65
	}
	return func() tea.Msg {
		manifest, err := LoadManifest(deployDir)
		if err != nil {
			return asyncDoneMsg{err: err}
		}
		for _, spec := range manifest.Sites {
			if err := applyThresholdToSite(spec, deployDir, temp); err != nil {
				return asyncDoneMsg{err: err}
			}
		}
		if err := markComplete(deployDir); err != nil {
			return asyncDoneMsg{err: err}
		}
		return asyncDoneMsg{body: fmt.Sprintf("Threshold %.0f°C applied to %d site(s). Setup complete.", temp, len(manifest.Sites))}
	}
}

func (m model) View() string {
	th := m.theme
	var b strings.Builder
	b.WriteString(th.LogoMark.Render("//"))
	b.WriteString(" ")
	b.WriteString(th.Wordmark.Render("Equate"))
	b.WriteString("  ")
	b.WriteString(th.Eyebrow.Render("COLLECTOR SETUP"))
	b.WriteString("\n")
	b.WriteString(m.progressRail())
	b.WriteString("\n\n")
	if m.err != "" {
		b.WriteString(th.Error.Render("error: " + m.err))
		b.WriteString("\n\n")
	}
	if m.loading {
		b.WriteString(th.Spinner.Render(m.spinner.View()))
		b.WriteString(" ")
		switch m.step {
		case stepStart:
			b.WriteString(th.Muted.Render("Building image and waiting for all site collectors…"))
		case stepReview:
			b.WriteString(th.Muted.Render("Scanning each site CIDR and accepting devices…"))
		case stepThresholds:
			b.WriteString(th.Muted.Render("Applying threshold to all sites…"))
		default:
			b.WriteString(th.Muted.Render("working…"))
		}
		return b.String()
	}
	switch m.step {
	case stepEnv:
		b.WriteString(th.Title.Render("Step 1 — Shared environment"))
		b.WriteString("\n\n")
		for i, input := range m.envInputs {
			b.WriteString(th.Label.Render(fmt.Sprintf("%d", i+1)))
			b.WriteString(" ")
			b.WriteString(input.View())
			b.WriteString("\n")
		}
		b.WriteString(th.Muted.Render("tab next field · enter continue"))
	case stepSites:
		b.WriteString(th.Title.Render("Step 2 — Site containers"))
		b.WriteString("\n\n")
		b.WriteString(th.Label.Render("count"))
		b.WriteString(" ")
		b.WriteString(m.siteCountInput.View())
		b.WriteString("\n")
		for i := range m.cidrInputs {
			b.WriteString(th.Label.Render("site id"))
			b.WriteString(" ")
			b.WriteString(m.siteIDInputs[i].View())
			b.WriteString("  ")
			b.WriteString(th.Label.Render("cidr"))
			b.WriteString(" ")
			b.WriteString(m.cidrInputs[i].View())
			b.WriteString("\n")
		}
		if m.body != "" {
			b.WriteString("\n")
			b.WriteString(th.Value.Render(m.body))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(th.Muted.Render("tab next field · enter on last CIDR to save artifacts"))
	case stepStart, stepReview:
		if m.step == stepStart {
			b.WriteString(th.Title.Render("Step 3 — Starting collectors"))
			b.WriteString("\n\n")
			if m.body != "" {
				b.WriteString(th.Value.Render(m.body))
				b.WriteString("\n\n")
			}
			b.WriteString(th.Muted.Render("enter to build and start all site containers"))
		} else {
			b.WriteString(th.Title.Render("Step 4 — Review inventory"))
			b.WriteString("\n")
			if m.body != "" {
				b.WriteString(th.Value.Render(m.body))
				b.WriteString("\n")
			}
			if m.err != "" {
				b.WriteString("\n")
				b.WriteString(th.Muted.Render("r retry discovery · s skip and continue"))
			} else if !m.loading {
				b.WriteString("\n")
				b.WriteString(th.Muted.Render("enter to run discovery for each site"))
			}
		}
	case stepThresholds:
		b.WriteString(th.Title.Render("Step 5 — Thresholds"))
		b.WriteString("\n\n")
		b.WriteString(th.Label.Render("Global temperature warning °C"))
		b.WriteString(" ")
		b.WriteString(m.thresholdInput.View())
		b.WriteString("\n\n")
		b.WriteString(th.Muted.Render("enter to apply to all sites and finish setup"))
	case stepDone:
		b.WriteString(th.Title.Render("Setup complete"))
		b.WriteString("\n\n")
		b.WriteString(th.Value.Render(m.body))
		b.WriteString("\n\n")
		b.WriteString(th.Muted.Render("Per-site operator TUI examples:"))
		b.WriteString("\n")
		if manifest, err := LoadManifest(m.deployDir); err == nil {
			for _, spec := range manifest.Sites {
				b.WriteString(th.Muted.Render(fmt.Sprintf(
					"docker compose exec -it %s /collector tui -socket /run/snmp-collector/control.sock -theme auto",
					spec.ServiceName,
				)))
				b.WriteString("\n")
			}
		}
		b.WriteString(th.Muted.Render("press q to quit"))
	}
	return b.String()
}

func (m model) progressRail() string {
	labels := []string{"Env", "Sites", "Start", "Review", "Thresholds", "Done"}
	parts := make([]string, len(labels))
	for i, label := range labels {
		if step(i) <= m.step {
			parts[i] = m.theme.TabActive.Render(label)
		} else {
			parts[i] = m.theme.TabIdle.Render(label)
		}
	}
	return strings.Join(parts, "  ·  ")
}
