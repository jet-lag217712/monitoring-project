package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/equate/ogsd/services/snmp-collector/internal/control"
)

type view int

const (
	viewInventory view = iota
	viewDiscovery
	viewTransit
	viewSettings
)

const autoRefreshInterval = 5 * time.Second

type model struct {
	client *control.Client
	theme  Theme
	view   view
	body   string
	err    string
	width  int
	height int

	siteID      string
	collectorID string
	revision    string
	deviceIDs   []string

	loading     bool
	lastUpdated time.Time
	spinner     spinner.Model
	viewport    viewport.Model
	ready       bool

	// mutation confirm state
	pendingToken    string
	pendingRevision string
	pendingAction   string
	confirmPrompt   string

	// threshold / dependency input
	inputMode   string // "", "threshold", "deps"
	textInput   textinput.Model
	inputDevice string
}

type refreshMsg struct {
	body        string
	err         string
	siteID      string
	collectorID string
	revision    string
	deviceIDs   []string
}

type pendingPreparedMsg struct {
	token    string
	revision string
	action   string
}

type tickMsg time.Time

func newModel(client *control.Client, theme Theme) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = theme.Spinner

	ti := textinput.New()
	ti.Placeholder = ""
	ti.CharLimit = 128
	ti.Width = 40

	return model{
		client:    client,
		theme:     theme,
		view:      viewInventory,
		spinner:   sp,
		textInput: ti,
		viewport:  viewport.New(80, 20),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.refresh(), tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(autoRefreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) refresh() tea.Cmd {
	view := m.view
	client := m.client
	theme := m.theme
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		body, meta, err := fetchView(ctx, client, theme, view)
		if err != nil {
			return refreshMsg{err: err.Error()}
		}
		return refreshMsg{
			body:        body,
			siteID:      meta.siteID,
			collectorID: meta.collectorID,
			revision:    meta.revision,
			deviceIDs:   meta.deviceIDs,
		}
	}
}

type viewMeta struct {
	siteID      string
	collectorID string
	revision    string
	deviceIDs   []string
}

func fetchView(ctx context.Context, client *control.Client, th Theme, v view) (string, viewMeta, error) {
	var meta viewMeta
	switch v {
	case viewInventory:
		resp, err := client.Call(ctx, "1", "inventory.list", nil)
		if err != nil {
			return "", meta, err
		}
		if !resp.OK {
			return "", meta, fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message)
		}
		meta.revision = fmt.Sprint(resp.Result["config_revision"])
		meta.deviceIDs = extractDeviceIDs(resp.Result)
		populateMeta(ctx, client, &meta)
		return formatInventory(th, resp.Result), meta, nil
	case viewDiscovery:
		status, err := client.Call(ctx, "1", "discovery.status", nil)
		if err != nil {
			return "", meta, err
		}
		if !status.OK {
			return "", meta, fmt.Errorf("%s: %s", status.Error.Code, status.Error.Message)
		}
		candidates, err := client.Call(ctx, "2", "discovery.candidates.list", nil)
		if err != nil {
			return "", meta, err
		}
		if !candidates.OK {
			return "", meta, fmt.Errorf("%s: %s", candidates.Error.Code, candidates.Error.Message)
		}
		populateMeta(ctx, client, &meta)
		return formatDiscoveryView(th, status.Result, candidates.Result), meta, nil
	case viewTransit:
		resp, err := client.Call(ctx, "1", "transport.get", nil)
		if err != nil {
			return "", meta, err
		}
		if !resp.OK {
			return "", meta, fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message)
		}
		meta.revision = fmt.Sprint(resp.Result["config_revision"])
		populateMeta(ctx, client, &meta)
		return formatTransit(th, resp.Result), meta, nil
	case viewSettings:
		resp, err := client.Call(ctx, "1", "config.get", nil)
		if err != nil {
			return "", meta, err
		}
		if !resp.OK {
			return "", meta, fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message)
		}
		meta.siteID = fmt.Sprint(resp.Result["site_id"])
		meta.collectorID = fmt.Sprint(resp.Result["collector_id"])
		meta.revision = fmt.Sprint(resp.Result["config_revision"])
		return formatConfig(th, resp.Result), meta, nil
	default:
		return "", meta, fmt.Errorf("unknown view")
	}
}

func populateMeta(ctx context.Context, client *control.Client, meta *viewMeta) {
	resp, err := client.Call(ctx, "meta", "config.get", nil)
	if err != nil || !resp.OK {
		return
	}
	if meta.siteID == "" {
		meta.siteID = fmt.Sprint(resp.Result["site_id"])
	}
	if meta.collectorID == "" {
		meta.collectorID = fmt.Sprint(resp.Result["collector_id"])
	}
}

func extractDeviceIDs(result map[string]any) []string {
	devices, _ := result["devices"].([]any)
	ids := make([]string, 0, len(devices))
	for _, raw := range devices {
		d, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if id, ok := d["id"].(string); ok && id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func (m model) firstDeviceID() string {
	if len(m.deviceIDs) == 0 {
		return ""
	}
	return m.deviceIDs[0]
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		chromeHeight := m.chromeHeight()
		footerHeight := 3
		if !m.ready {
			m.viewport = viewport.New(msg.Width, max(1, msg.Height-chromeHeight-footerHeight))
			m.viewport.YPosition = 0
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = max(1, msg.Height-chromeHeight-footerHeight)
		}
		m.viewport.SetContent(m.body)
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tickMsg:
		cmds = append(cmds, tickCmd())
		if m.confirmPrompt == "" && m.inputMode == "" {
			m.loading = true
			cmds = append(cmds, m.refresh())
		}
		return m, tea.Batch(cmds...)

	case refreshMsg:
		m.loading = false
		if msg.err != "" {
			m.err = msg.err
		} else {
			m.err = ""
			m.body = msg.body
			m.lastUpdated = time.Now()
			m.viewport.SetContent(m.body)
		}
		if msg.siteID != "" {
			m.siteID = msg.siteID
		}
		if msg.collectorID != "" {
			m.collectorID = msg.collectorID
		}
		if msg.revision != "" && msg.revision != "<nil>" {
			m.revision = msg.revision
		}
		if len(msg.deviceIDs) > 0 {
			m.deviceIDs = msg.deviceIDs
		}
		return m, nil

	case pendingPreparedMsg:
		m.pendingToken = msg.token
		m.pendingRevision = msg.revision
		m.pendingAction = msg.action
		m.confirmPrompt = "commit"
		m.err = ""
		m.inputMode = ""
		m.textInput.Blur()
		return m, nil

	case tea.KeyMsg:
		if m.confirmPrompt != "" {
			return m.handleConfirm(msg)
		}
		if m.inputMode != "" {
			return m.handleInput(msg)
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.view = viewInventory
			m.loading = true
			return m, m.refresh()
		case "2":
			m.view = viewDiscovery
			m.loading = true
			return m, m.refresh()
		case "3":
			m.view = viewTransit
			m.loading = true
			return m, m.refresh()
		case "4":
			m.view = viewSettings
			m.loading = true
			return m, m.refresh()
		case "tab", "right", "l":
			m.view = (m.view + 1) % 4
			m.loading = true
			return m, m.refresh()
		case "shift+tab", "left", "h":
			m.view = (m.view + 3) % 4
			m.loading = true
			return m, m.refresh()
		case "r":
			m.loading = true
			return m, m.refresh()
		case "R":
			m.loading = true
			return m, m.reload()
		case "t":
			return m.beginThresholdInput()
		case "S":
			if m.view == viewDiscovery {
				m.loading = true
				return m, m.discoveryScan()
			}
		case "A":
			if m.view == viewDiscovery {
				m.loading = true
				return m, m.discoveryAccept()
			}
		case "e", "E":
			if m.view == viewDiscovery {
				return m.beginDiscoveryPolicyInput()
			}
		case "d":
			if m.view == viewInventory {
				return m.beginDepsInput()
			}
		case "up", "k":
			m.viewport.LineUp(1)
			return m, nil
		case "down", "j":
			m.viewport.LineDown(1)
			return m, nil
		case "pgup":
			m.viewport.ViewUp()
			return m, nil
		case "pgdown":
			m.viewport.ViewDown()
			return m, nil
		}
	}

	return m, nil
}

func (m model) handleConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		prompt := m.confirmPrompt
		m.confirmPrompt = ""
		if prompt == "commit" {
			m.loading = true
			return m, m.commitPending()
		}
		return m, nil
	case "n", "N", "esc":
		m.confirmPrompt = ""
		m.pendingToken = ""
		m.pendingRevision = ""
		m.pendingAction = ""
		m.err = "mutation cancelled"
		return m, nil
	}
	return m, nil
}

func (m model) beginThresholdInput() (tea.Model, tea.Cmd) {
	m.inputMode = "threshold"
	m.inputDevice = ""
	m.textInput.SetValue("")
	m.textInput.Placeholder = "temperature °C (e.g. 65)"
	m.textInput.Focus()
	m.err = ""
	return m, textinput.Blink
}

func (m model) beginDepsInput() (tea.Model, tea.Cmd) {
	id := m.firstDeviceID()
	if id == "" {
		m.err = "no device selected"
		return m, nil
	}
	m.inputMode = "deps"
	m.inputDevice = id
	m.textInput.SetValue("")
	m.textInput.Placeholder = "upstream ids, comma-separated (empty clears)"
	m.textInput.Focus()
	m.err = ""
	return m, textinput.Blink
}

func (m model) handleInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.inputMode = ""
		m.textInput.Blur()
		m.err = "edit cancelled"
		return m, nil
	case "enter":
		mode := m.inputMode
		value := strings.TrimSpace(m.textInput.Value())
		device := m.inputDevice
		m.inputMode = ""
		m.textInput.Blur()
		m.loading = true
		if mode == "threshold" {
			return m, m.prepareThreshold(value, device)
		}
		if mode == "deps" {
			return m, m.prepareDependencies(value, device)
		}
		if mode == "discovery-policy" {
			return m, m.prepareDiscoveryPolicy(value)
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) prepareThreshold(value, deviceID string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		temp := 65.0
		if value != "" {
			parsed, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return refreshMsg{err: "temperature must be a number"}
			}
			temp = parsed
		} else {
			cfg, err := client.Call(ctx, "p1", "config.get", nil)
			if err != nil {
				return refreshMsg{err: err.Error()}
			}
			if !cfg.OK {
				return refreshMsg{err: cfg.Error.Code + ": " + cfg.Error.Message}
			}
			health, _ := cfg.Result["health"].(map[string]any)
			if health != nil {
				if v, ok := health["temperature_warning_c"].(float64); ok {
					temp = v
				}
			}
		}

		params := map[string]any{"temperature_warning_c": temp}
		if deviceID != "" {
			params["device_id"] = deviceID
		}
		resp, err := client.Call(ctx, "p2", "thresholds.prepare", params)
		if err != nil {
			return refreshMsg{err: err.Error()}
		}
		if !resp.OK {
			return refreshMsg{err: resp.Error.Code + ": " + resp.Error.Message}
		}
		return pendingPreparedMsg{
			token:    fmt.Sprint(resp.Result["confirm_token"]),
			revision: fmt.Sprint(resp.Result["revision"]),
			action:   "thresholds",
		}
	}
}

func (m model) beginDiscoveryPolicyInput() (tea.Model, tea.Cmd) {
	m.inputMode = "discovery-policy"
	m.textInput.SetValue("")
	m.textInput.Placeholder = "CIDRs comma-separated (e.g. 10.255.0.0/24)"
	m.textInput.EchoMode = textinput.EchoNormal
	m.textInput.Focus()
	m.err = ""
	return m, textinput.Blink
}

func (m model) discoveryScan() tea.Cmd {
	client := m.client
	view := m.view
	theme := m.theme
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if _, err := client.Call(ctx, "ds1", "discovery.scan.start", nil); err != nil {
			return refreshMsg{err: err.Error()}
		}
		body, _, err := fetchView(ctx, client, theme, view)
		if err != nil {
			return refreshMsg{err: err.Error()}
		}
		return refreshMsg{body: body}
	}
}

func (m model) discoveryAccept() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		list, err := client.Call(ctx, "da1", "discovery.candidates.list", nil)
		if err != nil {
			return refreshMsg{err: err.Error()}
		}
		if !list.OK {
			return refreshMsg{err: list.Error.Code + ": " + list.Error.Message}
		}
		raw, _ := list.Result["candidates"].([]any)
		reviews := make([]map[string]any, 0)
		for _, item := range raw {
			c, ok := item.(map[string]any)
			if !ok || fmt.Sprint(c["result"]) != "success" {
				continue
			}
			ip := fmt.Sprint(c["ip"])
			id := "discovered-" + strings.ReplaceAll(ip, ".", "-")
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
					"community_env": "SNMP_COMMUNITY",
					"version":       "2c",
				},
			})
		}
		if len(reviews) == 0 {
			return refreshMsg{err: "no successful candidates to accept"}
		}
		prepare, err := client.Call(ctx, "da2", "discovery.accept.prepare", map[string]any{"reviews": reviews})
		if err != nil {
			return refreshMsg{err: err.Error()}
		}
		if !prepare.OK {
			return refreshMsg{err: prepare.Error.Code + ": " + prepare.Error.Message}
		}
		return pendingPreparedMsg{
			token:    fmt.Sprint(prepare.Result["confirm_token"]),
			revision: fmt.Sprint(prepare.Result["revision"]),
			action:   "discovery.accept",
		}
	}
}

func (m model) prepareDiscoveryPolicy(value string) tea.Cmd {
	client := m.client
	cidrs := []string{}
	for _, part := range strings.Split(value, ",") {
		c := strings.TrimSpace(part)
		if c != "" {
			cidrs = append(cidrs, c)
		}
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := client.Call(ctx, "dp1", "discovery.policy.prepare", map[string]any{
			"allowed_cidrs":         cidrs,
			"community_env":         "SNMP_DISCOVERY_COMMUNITY",
			"max_probes_per_second": 5.0,
			"probe_burst":           2,
		})
		if err != nil {
			return refreshMsg{err: err.Error()}
		}
		if !resp.OK {
			return refreshMsg{err: resp.Error.Code + ": " + resp.Error.Message}
		}
		return pendingPreparedMsg{
			token:    fmt.Sprint(resp.Result["confirm_token"]),
			revision: fmt.Sprint(resp.Result["revision"]),
			action:   "discovery.policy",
		}
	}
}

func (m model) prepareDependencies(value, deviceID string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		upstreams := []string{}
		if value != "" {
			for _, part := range strings.Split(value, ",") {
				id := strings.TrimSpace(part)
				if id != "" {
					upstreams = append(upstreams, id)
				}
			}
		}
		resp, err := client.Call(ctx, "d1", "dependencies.prepare", map[string]any{
			"device_id":           deviceID,
			"upstream_device_ids": upstreams,
		})
		if err != nil {
			return refreshMsg{err: err.Error()}
		}
		if !resp.OK {
			return refreshMsg{err: resp.Error.Code + ": " + resp.Error.Message}
		}
		return pendingPreparedMsg{
			token:    fmt.Sprint(resp.Result["confirm_token"]),
			revision: fmt.Sprint(resp.Result["revision"]),
			action:   "dependencies",
		}
	}
}

func (m model) commitPending() tea.Cmd {
	client := m.client
	token := m.pendingToken
	revision := m.pendingRevision
	action := m.pendingAction
	theme := m.theme
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		method := action + ".commit"
		resp, err := client.Call(ctx, "c1", method, map[string]any{
			"confirm_token": token,
			"revision":      revision,
		})
		if err != nil {
			return refreshMsg{err: err.Error()}
		}
		if !resp.OK {
			return refreshMsg{err: resp.Error.Code + ": " + resp.Error.Message}
		}
		reload, err := client.Call(ctx, "c2", "config.reload", nil)
		if err != nil {
			return refreshMsg{err: err.Error()}
		}
		if !reload.OK {
			return refreshMsg{err: reload.Error.Code + ": " + reload.Error.Message}
		}
		return refreshMsg{
			body: "Mutation committed and configuration reloaded.\n\n" + formatReloadResult(theme, reload.Result),
		}
	}
}

func (m model) reload() tea.Cmd {
	client := m.client
	theme := m.theme
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resp, err := client.Call(ctx, "r1", "config.reload", nil)
		if err != nil {
			return refreshMsg{err: err.Error()}
		}
		if !resp.OK {
			return refreshMsg{err: resp.Error.Code + ": " + resp.Error.Message}
		}
		return refreshMsg{body: formatReloadResult(theme, resp.Result)}
	}
}

func (m model) renderChrome() string {
	th := m.theme
	var b strings.Builder
	b.WriteString(renderHeader(th, m.siteID, m.collectorID, m.revision, m.lastUpdated, m.loading))
	if m.loading {
		b.WriteString(th.Spinner.Render(m.spinner.View()))
		b.WriteString("\n")
	}
	b.WriteString(renderTabs(th, m.view))
	b.WriteString("\n")
	b.WriteString(th.Muted.Render(strings.Repeat("─", max(8, min(m.width, 100)))))
	b.WriteString("\n")
	return b.String()
}

func (m model) View() string {
	th := m.theme
	var b strings.Builder

	chrome := m.renderChrome()
	b.WriteString(chrome)

	if m.confirmPrompt != "" {
		b.WriteString(renderConfirm(th, m.pendingAction, m.pendingRevision))
		b.WriteString("\n")
		b.WriteString(renderHelp(th))
		return b.String()
	}

	if m.inputMode != "" {
		prompt := "Temperature warning (°C)"
		if m.inputMode == "discovery-policy" {
			prompt = "Discovery CIDR allowlist"
		} else if m.inputMode == "deps" {
			prompt = "Upstream device IDs for " + m.inputDevice
		} else if m.inputDevice != "" {
			prompt = "Temperature warning (°C) for " + m.inputDevice
		}
		b.WriteString(th.Confirm.Render(prompt))
		b.WriteString("\n")
		b.WriteString(m.textInput.View())
		b.WriteString("\n")
		b.WriteString(th.Muted.Render("enter submit  ·  esc cancel"))
		return b.String()
	}

	if m.err != "" {
		b.WriteString(renderError(th, m.err))
		b.WriteString("\n")
	}

	if m.ready {
		b.WriteString(m.viewport.View())
	} else {
		b.WriteString(m.body)
	}
	b.WriteString("\n")
	if actions := renderActions(th, m.view); actions != "" {
		b.WriteString(actions)
		b.WriteString("\n")
	}
	b.WriteString(renderHelp(th))
	return b.String()
}

func (m model) chromeHeight() int {
	return lipgloss.Height(m.renderChrome())
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
