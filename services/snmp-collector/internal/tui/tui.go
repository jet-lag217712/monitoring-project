// Package tui is the Bubble Tea local operator client for the control socket.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/equate/ogsd/services/snmp-collector/internal/control"
)

type view int

const (
	viewInventory view = iota
	viewDevice
	viewDiscovery
	viewThresholds
	viewTransport
	viewConfig
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	helpStyle  = lipgloss.NewStyle().Faint(true)
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

// Run starts the interactive TUI against the given control socket.
func Run(socketPath string) error {
	client := control.NewClient(socketPath)
	m := model{
		client: client,
		view:   viewInventory,
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

type model struct {
	client *control.Client
	view   view
	body   string
	err    string
	width  int
	height int

	// mutation confirm state
	pendingToken    string
	pendingRevision string
	pendingAction   string
	confirmPrompt   string
}

type refreshMsg struct {
	body string
	err  string
}

func (m model) Init() tea.Cmd {
	return m.refresh()
}

func (m model) refresh() tea.Cmd {
	view := m.view
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		body, err := fetchView(ctx, client, view)
		if err != nil {
			return refreshMsg{err: err.Error()}
		}
		return refreshMsg{body: body}
	}
}

func fetchView(ctx context.Context, client *control.Client, v view) (string, error) {
	switch v {
	case viewInventory:
		resp, err := client.Call(ctx, "1", "inventory.list", nil)
		if err != nil {
			return "", err
		}
		if !resp.OK {
			return "", fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message)
		}
		return formatResult("Inventory", resp.Result), nil
	case viewDevice:
		list, err := client.Call(ctx, "1", "inventory.list", nil)
		if err != nil {
			return "", err
		}
		if !list.OK {
			return "", fmt.Errorf("%s: %s", list.Error.Code, list.Error.Message)
		}
		devices, _ := list.Result["devices"].([]any)
		if len(devices) == 0 {
			return "No devices configured.", nil
		}
		first, _ := devices[0].(map[string]any)
		id, _ := first["id"].(string)
		resp, err := client.Call(ctx, "2", "device.get", map[string]any{"device_id": id})
		if err != nil {
			return "", err
		}
		if !resp.OK {
			return "", fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message)
		}
		return formatResult("Device "+id, resp.Result), nil
	case viewDiscovery:
		resp, err := client.Call(ctx, "1", "discovery.status", nil)
		if err != nil {
			return "", err
		}
		if !resp.OK {
			return "", fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message)
		}
		return formatResult("Discovery", resp.Result), nil
	case viewThresholds:
		resp, err := client.Call(ctx, "1", "config.get", nil)
		if err != nil {
			return "", err
		}
		if !resp.OK {
			return "", fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message)
		}
		return formatResult("Thresholds (press t to prepare global edit)", resp.Result), nil
	case viewTransport:
		resp, err := client.Call(ctx, "1", "transport.get", nil)
		if err != nil {
			return "", err
		}
		if !resp.OK {
			return "", fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message)
		}
		return formatResult("Transport", resp.Result), nil
	case viewConfig:
		resp, err := client.Call(ctx, "1", "config.get", nil)
		if err != nil {
			return "", err
		}
		if !resp.OK {
			return "", fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message)
		}
		return formatResult("Configuration", resp.Result), nil
	default:
		return "", fmt.Errorf("unknown view")
	}
}

func formatResult(title string, result map[string]any) string {
	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")
	for _, key := range sortedKeys(result) {
		fmt.Fprintf(&b, "%s: %v\n", key, result[key])
	}
	return b.String()
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case refreshMsg:
		m.body = msg.body
		m.err = msg.err
		return m, nil
	case pendingPreparedMsg:
		m.pendingToken = msg.token
		m.pendingRevision = msg.revision
		m.pendingAction = msg.action
		m.confirmPrompt = "commit"
		m.err = ""
		return m, nil
	case tea.KeyMsg:
		if m.confirmPrompt != "" {
			return m.handleConfirm(msg)
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.view = viewInventory
			return m, m.refresh()
		case "2":
			m.view = viewDevice
			return m, m.refresh()
		case "3":
			m.view = viewDiscovery
			return m, m.refresh()
		case "4":
			m.view = viewThresholds
			return m, m.refresh()
		case "5":
			m.view = viewTransport
			return m, m.refresh()
		case "6":
			m.view = viewConfig
			return m, m.refresh()
		case "r":
			return m, m.refresh()
		case "R":
			return m, m.reload()
		case "t":
			return m, m.prepareThreshold()
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

func (m model) prepareThreshold() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cfg, err := client.Call(ctx, "p1", "config.get", nil)
		if err != nil {
			return refreshMsg{err: err.Error()}
		}
		if !cfg.OK {
			return refreshMsg{err: cfg.Error.Code + ": " + cfg.Error.Message}
		}
		health, _ := cfg.Result["health"].(map[string]any)
		current := 65.0
		if health != nil {
			if v, ok := health["temperature_warning_c"].(float64); ok {
				current = v
			}
		}
		resp, err := client.Call(ctx, "p2", "thresholds.prepare", map[string]any{
			"temperature_warning_c": current,
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
			action:   "thresholds",
		}
	}
}

type pendingPreparedMsg struct {
	token    string
	revision string
	action   string
}

func (m model) commitPending() tea.Cmd {
	client := m.client
	token := m.pendingToken
	revision := m.pendingRevision
	action := m.pendingAction
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
		return refreshMsg{body: "Mutation committed and configuration reloaded.\n\n" + formatResult("Reload", reload.Result)}
	}
}

func (m model) reload() tea.Cmd {
	client := m.client
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
		return refreshMsg{body: formatResult("Reload", resp.Result)}
	}
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("SNMP Collector Operator TUI"))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("1 inventory  2 device  3 discovery  4 thresholds  5 transport  6 config  r refresh  R reload  t prepare-threshold  q quit"))
	b.WriteString("\n\n")
	if m.confirmPrompt != "" {
		b.WriteString("Confirm mutation commit? [y/n]\n")
		b.WriteString(fmt.Sprintf("action=%s revision=%s\n", m.pendingAction, m.pendingRevision))
		return b.String()
	}
	if m.err != "" {
		b.WriteString(errStyle.Render("error: " + m.err))
		b.WriteString("\n\n")
	}
	b.WriteString(m.body)
	return b.String()
}
