package setup

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/equate/ogsd/services/snmp-collector/internal/tui"
)

func (m model) firstStepAfterSplash() step {
	if m.profileCfg.RequireAdminUser {
		return stepAdminUser
	}
	return stepEnv
}

func (m model) updateAdminFocus() model {
	inputs := []*textinput.Model{&m.adminUsernameInput, &m.adminPasswordInput, &m.adminConfirmInput}
	for i, input := range inputs {
		if i == m.adminFocus {
			input.Focus()
		} else {
			input.Blur()
		}
	}
	return m
}

func (m model) updateAdminUser(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.adminPhase == "loading" {
		return m, nil
	}
	if m.adminPhase == "choose" {
		key, ok := msg.(tea.KeyMsg)
		if !ok {
			return m, nil
		}
		switch key.String() {
		case "1":
			return m.advanceFromAdminStep("Using existing appliance administrator accounts.")
		case "2":
			m.adminPhase = "create"
			m.err = ""
			m.body = ""
			m.adminUsernameInput.SetValue("")
			m.adminPasswordInput.SetValue("")
			m.adminConfirmInput.SetValue("")
			m.adminFocus = 0
			return m.updateAdminFocus(), textinput.Blink
		}
		return m, nil
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		switch m.adminFocus {
		case 0:
			m.adminUsernameInput, cmd = m.adminUsernameInput.Update(msg)
		case 1:
			m.adminPasswordInput, cmd = m.adminPasswordInput.Update(msg)
		default:
			m.adminConfirmInput, cmd = m.adminConfirmInput.Update(msg)
		}
		return m, cmd
	}
	switch key.String() {
	case "tab", "down":
		m.adminFocus = (m.adminFocus + 1) % 3
		return m.updateAdminFocus(), textinput.Blink
	case "shift+tab", "up":
		m.adminFocus = (m.adminFocus - 1 + 3) % 3
		return m.updateAdminFocus(), textinput.Blink
	case "enter":
		if m.adminFocus < 2 {
			m.adminFocus++
			return m.updateAdminFocus(), textinput.Blink
		}
		m.loading = true
		m.err = ""
		return m, m.createInitialAdmin()
	}
	var cmd tea.Cmd
	switch m.adminFocus {
	case 0:
		m.adminUsernameInput, cmd = m.adminUsernameInput.Update(msg)
	case 1:
		m.adminPasswordInput, cmd = m.adminPasswordInput.Update(msg)
	default:
		m.adminConfirmInput, cmd = m.adminConfirmInput.Update(msg)
	}
	return m, cmd
}

func (m model) createInitialAdmin() tea.Cmd {
	deployDir := m.deployDir
	username := strings.TrimSpace(m.adminUsernameInput.Value())
	password := m.adminPasswordInput.Value()
	confirm := m.adminConfirmInput.Value()
	return func() tea.Msg {
		if username == "" || password == "" {
			return asyncDoneMsg{err: fmt.Errorf("initial administrator username and password are required")}
		}
		if password != confirm {
			return asyncDoneMsg{err: fmt.Errorf("password confirmation does not match")}
		}
		helper, err := resolvePamHelper(deployDir)
		if err != nil {
			return asyncDoneMsg{err: err}
		}
		if err := pamUserCreate(helper, username, password); err != nil {
			return asyncDoneMsg{err: err}
		}
		return asyncDoneMsg{body: fmt.Sprintf("Created initial administrator %q.", username)}
	}
}

func (m model) loadExistingAdmins() tea.Cmd {
	deployDir := m.deployDir
	return func() tea.Msg {
		helper, err := resolvePamHelper(deployDir)
		if err != nil {
			return existingAdminsMsg{err: err}
		}
		hasExisting, body, err := pamHasExistingUsers(helper)
		if err != nil {
			return existingAdminsMsg{err: err}
		}
		return existingAdminsMsg{hasExisting: hasExisting, body: body}
	}
}

func (m model) advanceFromAdminStep(body string) (model, tea.Cmd) {
	m.err = ""
	m.body = body
	if m.profileCfg.UserManagement {
		m.step = stepUsers
		m.usersMode = "menu"
		return m, m.refreshUsersList()
	}
	m.step = stepEnv
	m.envFocus = 0
	m = m.updateEnvFocus()
	return m, textinput.Blink
}

func (m model) updateUsers(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.usersMode == "menu" {
		key, ok := msg.(tea.KeyMsg)
		if !ok {
			return m, nil
		}
		switch key.String() {
		case "1":
			m.usersMode = "create"
			m.usersFocus = 0
			return m.updateUsersFocus(), textinput.Blink
		case "2":
			m.loading = true
			return m, m.refreshUsersList()
		case "3":
			m.usersMode = "disable"
			m.usersFocus = 0
			return m.updateUsersFocus(), textinput.Blink
		case "4":
			m.usersMode = "reset"
			m.usersFocus = 0
			return m.updateUsersFocus(), textinput.Blink
		case "5", "c", "enter":
			m.step = stepEnv
			m.envFocus = 0
			m = m.updateEnvFocus()
			return m, textinput.Blink
		}
		return m, nil
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		switch m.usersFocus {
		case 0:
			m.usersUsername, cmd = m.usersUsername.Update(msg)
		case 1:
			m.usersPassword, cmd = m.usersPassword.Update(msg)
		default:
			m.usersConfirm, cmd = m.usersConfirm.Update(msg)
		}
		return m, cmd
	}
	if key.String() == "esc" {
		m.usersMode = "menu"
		m.usersUsername.SetValue("")
		m.usersPassword.SetValue("")
		m.usersConfirm.SetValue("")
		return m, m.refreshUsersList()
	}
	lastFocus := 1
	if m.usersMode == "create" || m.usersMode == "reset" {
		lastFocus = 2
	}
	switch key.String() {
	case "tab", "down":
		m.usersFocus = (m.usersFocus + 1) % (lastFocus + 1)
		return m.updateUsersFocus(), textinput.Blink
	case "shift+tab", "up":
		m.usersFocus = (m.usersFocus - 1 + lastFocus + 1) % (lastFocus + 1)
		return m.updateUsersFocus(), textinput.Blink
	case "enter":
		if m.usersFocus < lastFocus {
			m.usersFocus++
			return m.updateUsersFocus(), textinput.Blink
		}
		m.loading = true
		return m, m.runUsersAction()
	}
	var cmd tea.Cmd
	switch m.usersFocus {
	case 0:
		m.usersUsername, cmd = m.usersUsername.Update(msg)
	case 1:
		m.usersPassword, cmd = m.usersPassword.Update(msg)
	default:
		m.usersConfirm, cmd = m.usersConfirm.Update(msg)
	}
	return m, cmd
}

func (m model) updateUsersFocus() model {
	m.usersUsername.Blur()
	m.usersPassword.Blur()
	m.usersConfirm.Blur()
	switch m.usersFocus {
	case 0:
		m.usersUsername.Focus()
	case 1:
		m.usersPassword.Focus()
	default:
		m.usersConfirm.Focus()
	}
	return m
}

func (m model) refreshUsersList() tea.Cmd {
	deployDir := m.deployDir
	return func() tea.Msg {
		helper, err := resolvePamHelper(deployDir)
		if err != nil {
			return asyncDoneMsg{err: err}
		}
		out, err := pamUserList(helper)
		if err != nil {
			return asyncDoneMsg{err: err}
		}
		body := strings.TrimSpace(out)
		if body == "" {
			body = "No appliance users listed."
		}
		return asyncDoneMsg{body: body}
	}
}

func (m model) runUsersAction() tea.Cmd {
	deployDir := m.deployDir
	mode := m.usersMode
	username := strings.TrimSpace(m.usersUsername.Value())
	password := m.usersPassword.Value()
	confirm := m.usersConfirm.Value()
	return func() tea.Msg {
		helper, err := resolvePamHelper(deployDir)
		if err != nil {
			return asyncDoneMsg{err: err}
		}
		switch mode {
		case "create":
			if username == "" || password == "" {
				return asyncDoneMsg{err: fmt.Errorf("username and password are required")}
			}
			if password != confirm {
				return asyncDoneMsg{err: fmt.Errorf("password confirmation does not match")}
			}
			if err := pamUserCreate(helper, username, password); err != nil {
				return asyncDoneMsg{err: err}
			}
			return asyncDoneMsg{body: fmt.Sprintf("Created user %q.", username)}
		case "disable":
			if username == "" {
				return asyncDoneMsg{err: fmt.Errorf("username is required")}
			}
			if err := pamUserDisable(helper, username); err != nil {
				return asyncDoneMsg{err: err}
			}
			return asyncDoneMsg{body: fmt.Sprintf("Disabled user %q.", username)}
		case "reset":
			if username == "" || password == "" {
				return asyncDoneMsg{err: fmt.Errorf("username and password are required")}
			}
			if password != confirm {
				return asyncDoneMsg{err: fmt.Errorf("password confirmation does not match")}
			}
			if err := pamUserReset(helper, username, password); err != nil {
				return asyncDoneMsg{err: err}
			}
			return asyncDoneMsg{body: fmt.Sprintf("Reset password for %q.", username)}
		default:
			return asyncDoneMsg{err: fmt.Errorf("unknown users action")}
		}
	}
}

func (m model) updateReviewManual(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.reviewPhase == "" || m.reviewPhase == "scan" {
		key, ok := msg.(tea.KeyMsg)
		if !ok {
			return m, nil
		}
		if key.String() == "enter" && !m.loading {
			m.loading = true
			m.err = ""
			return m, m.runReviewScanSite()
		}
		if key.String() == "s" || key.String() == "S" {
			return m, m.skipReviewSite()
		}
		return m, nil
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "up", "k":
		if m.reviewCursor > 0 {
			m.reviewCursor--
			m = m.syncReviewScroll()
		}
		return m, nil
	case "down", "j":
		if m.reviewCursor < len(m.reviewCandidates)-1 {
			m.reviewCursor++
			m = m.syncReviewScroll()
		}
		return m, nil
	case " ":
		if len(m.reviewApproved) > m.reviewCursor {
			m.reviewApproved[m.reviewCursor] = !m.reviewApproved[m.reviewCursor]
		}
		return m, nil
	case "enter":
		m.loading = true
		m.err = ""
		return m, m.runAcceptReviewSite()
	case "s", "S":
		return m, m.skipReviewSite()
	}
	return m, nil
}

func scanSiteCandidatesForSite(deployDir string, spec SiteSpec) (reviewScanMsg, error) {
	if err := loadEnvFile(envPath(deployDir)); err != nil {
		return reviewScanMsg{}, fmt.Errorf("load .env: %w", err)
	}
	client := newDeployControl(deployDir, spec)
	candidates, err := runDiscoveryScan(client)
	if err != nil {
		return reviewScanMsg{}, err
	}
	approved := make([]bool, len(candidates))
	for i, c := range candidates {
		approved[i] = fmt.Sprint(c["result"]) == "success"
	}
	body := fmt.Sprintf("%s: %d candidate(s) — toggle with space, enter to accept reviewed", spec.SiteID, len(candidates))
	if len(candidates) == 0 {
		body = fmt.Sprintf("%s: no candidates found", spec.SiteID)
	}
	return reviewScanMsg{candidates: candidates, approved: approved, body: body}, nil
}

func (m model) runReviewScanSite() tea.Cmd {
	deployDir := m.deployDir
	siteIdx := m.reviewSiteIdx
	sites := m.sites
	return func() tea.Msg {
		if len(sites) == 0 {
			manifest, err := LoadManifest(deployDir)
			if err != nil {
				return asyncDoneMsg{err: err}
			}
			sites = manifest.Sites
		}
		if siteIdx >= len(sites) {
			return asyncDoneMsg{err: fmt.Errorf("no site at index %d", siteIdx)}
		}
		msg, err := scanSiteCandidatesForSite(deployDir, sites[siteIdx])
		if err != nil {
			return asyncDoneMsg{err: err}
		}
		return msg
	}
}

type reviewScanMsg struct {
	candidates []map[string]any
	approved   []bool
	body       string
}

func (m model) runAcceptReviewSite() tea.Cmd {
	deployDir := m.deployDir
	siteIdx := m.reviewSiteIdx
	candidates := m.reviewCandidates
	approved := m.reviewApproved
	sites := m.sites
	return func() tea.Msg {
		if len(sites) == 0 {
			manifest, err := LoadManifest(deployDir)
			if err != nil {
				return asyncDoneMsg{err: err}
			}
			sites = manifest.Sites
		}
		if siteIdx >= len(sites) {
			return asyncDoneMsg{err: fmt.Errorf("invalid site index")}
		}
		spec := sites[siteIdx]
		client := newDeployControl(deployDir, spec)
		accepted, err := acceptApprovedCandidates(spec.ManagedInventoryPath(deployDir), "SNMP_COMMUNITY", client, candidates, approved)
		if err != nil {
			return asyncDoneMsg{err: fmt.Errorf("%s: %w", spec.SiteID, err)}
		}
		line := fmt.Sprintf("%s: accepted %d reviewed device(s)", spec.SiteID, accepted)
		return reviewAcceptMsg{line: line}
	}
}

type reviewAcceptMsg struct {
	line string
}

func (m model) skipReviewSite() tea.Cmd {
	siteIdx := m.reviewSiteIdx
	sites := m.sites
	return func() tea.Msg {
		siteID := fmt.Sprintf("site-%d", siteIdx+1)
		if siteIdx < len(sites) {
			siteID = sites[siteIdx].SiteID
		}
		return reviewAcceptMsg{line: fmt.Sprintf("%s: skipped discovery review", siteID)}
	}
}

func (m model) viewApplianceEnv(th tui.Theme) string {
	var body strings.Builder
	body.WriteString(th.Title.Render("Step 1 - SNMP communities"))
	body.WriteString("\n\n")
	body.WriteString(th.Muted.Render("MQTT uses the internal Mosquitto broker configured during VM install."))
	body.WriteString("\n\n")
	labels := []string{"community", "discovery"}
	for i, input := range m.envInputs {
		label := labels[i]
		body.WriteString(th.Label.Render(label))
		body.WriteString(" ")
		body.WriteString(input.View())
		body.WriteString("\n")
	}
	body.WriteString("\n")
	body.WriteString(th.Muted.Render("tab next field · enter continue"))
	return body.String()
}

func (m model) viewAdminUser(th tui.Theme) string {
	var body strings.Builder
	body.WriteString(th.Title.Render("Initial administrator"))
	body.WriteString("\n\n")
	if m.adminPhase == "loading" {
		body.WriteString(th.Muted.Render("Checking for existing appliance administrators…"))
		return body.String()
	}
	if m.adminPhase == "choose" {
		body.WriteString(th.Muted.Render("Appliance administrator account(s) already exist:"))
		body.WriteString("\n\n")
		body.WriteString(th.Value.Render(m.body))
		body.WriteString("\n\n")
		body.WriteString(th.Muted.Render("1 keep existing administrator(s)"))
		body.WriteString("\n")
		body.WriteString(th.Muted.Render("2 create a different administrator"))
		return body.String()
	}
	body.WriteString(th.Muted.Render("Create the first appliance administrator (no default password)."))
	body.WriteString("\n\n")
	body.WriteString(th.Label.Render("username"))
	body.WriteString(" ")
	body.WriteString(m.adminUsernameInput.View())
	body.WriteString("\n")
	body.WriteString(th.Label.Render("password"))
	body.WriteString(" ")
	body.WriteString(m.adminPasswordInput.View())
	body.WriteString("\n")
	body.WriteString(th.Label.Render("confirm"))
	body.WriteString(" ")
	body.WriteString(m.adminConfirmInput.View())
	body.WriteString("\n\n")
	if m.body != "" {
		body.WriteString(th.Value.Render(m.body))
		body.WriteString("\n\n")
	}
	body.WriteString(th.Muted.Render("tab next field · enter on confirm to create"))
	return body.String()
}

func (m model) viewUsers(th tui.Theme) string {
	var body strings.Builder
	body.WriteString(th.Title.Render("Appliance users (PAM)"))
	body.WriteString("\n\n")
	if m.usersMode == "menu" {
		if m.usersBody != "" {
			body.WriteString(th.Value.Render(m.usersBody))
			body.WriteString("\n\n")
		}
		body.WriteString(th.Muted.Render("1 create · 2 list · 3 disable · 4 reset password · 5 continue"))
		return body.String()
	}
	switch m.usersMode {
	case "create":
		body.WriteString(th.Label.Render("new username"))
	case "disable":
		body.WriteString(th.Label.Render("username to disable"))
	case "reset":
		body.WriteString(th.Label.Render("username to reset"))
	}
	body.WriteString(" ")
	body.WriteString(m.usersUsername.View())
	body.WriteString("\n")
	if m.usersMode == "create" || m.usersMode == "reset" {
		body.WriteString(th.Label.Render("password"))
		body.WriteString(" ")
		body.WriteString(m.usersPassword.View())
		body.WriteString("\n")
		body.WriteString(th.Label.Render("confirm"))
		body.WriteString(" ")
		body.WriteString(m.usersConfirm.View())
		body.WriteString("\n")
	}
	body.WriteString("\n")
	body.WriteString(th.Muted.Render("esc back to menu · enter on last field to submit"))
	return body.String()
}

func (m model) syncReviewScroll() model {
	visible := reviewListVisibleRows(m.height, m.applianceProgressRail() != "")
	m.reviewScrollTop = reviewScrollTopForCursor(
		m.reviewCursor,
		m.reviewScrollTop,
		len(m.reviewCandidates),
		visible,
	)
	return m
}

func (m model) viewReviewManual(th tui.Theme) string {
	var body strings.Builder
	body.WriteString(th.Title.Render("Step 4 - Review inventory (manual)"))
	body.WriteString("\n")
	siteLabel := fmt.Sprintf("site %d", m.reviewSiteIdx+1)
	if m.reviewSiteIdx < len(m.sites) {
		siteLabel = m.sites[m.reviewSiteIdx].SiteID
	}
	body.WriteString(th.Muted.Render("Reviewing " + siteLabel))
	body.WriteString("\n")
	if m.body != "" {
		body.WriteString(th.Value.Render(m.body))
		body.WriteString("\n")
	}
	if m.reviewPhase == "pick" && len(m.reviewCandidates) > 0 {
		visible := reviewListVisibleRows(m.height, m.applianceProgressRail() != "")
		total := len(m.reviewCandidates)
		scrollTop := reviewScrollTopForCursor(m.reviewCursor, m.reviewScrollTop, total, visible)
		end := scrollTop + visible
		if end > total {
			end = total
		}
		if scrollTop > 0 {
			body.WriteString(th.Muted.Render(fmt.Sprintf("  ↑ %d more above", scrollTop)))
			body.WriteString("\n")
		}
		for i := scrollTop; i < end; i++ {
			c := m.reviewCandidates[i]
			marker := "[ ]"
			if m.reviewApproved[i] {
				marker = "[x]"
			}
			prefix := "  "
			if i == m.reviewCursor {
				prefix = th.Confirm.Render("> ")
			}
			line := fmt.Sprintf("%s %s %s", marker, formatCandidateSummary(c), "")
			body.WriteString(prefix)
			body.WriteString(th.Value.Render(line))
			body.WriteString("\n")
		}
		remaining := total - end
		if remaining > 0 {
			body.WriteString(th.Muted.Render(fmt.Sprintf("  ↓ %d more below", remaining)))
			body.WriteString("\n")
		}
		body.WriteString("\n")
		if total > visible {
			body.WriteString(th.Muted.Render(fmt.Sprintf("showing %d–%d of %d · ", scrollTop+1, end, total)))
		}
		body.WriteString(th.Muted.Render("↑/↓ scroll · space toggle · enter accept reviewed · s skip site"))
	} else if !m.loading {
		body.WriteString("\n")
		body.WriteString(th.Muted.Render("enter scan site · s skip site"))
	}
	return body.String()
}

func (m model) applianceProgressLabels() []string {
	if m.profileCfg.RequireAdminUser {
		return []string{"Admin", "Users", "SNMP", "Sites", "Start", "Review", "Thresholds", "Done"}
	}
	return nil
}

func (m model) progressIndex() int {
	switch m.step {
	case stepAdminUser:
		return 0
	case stepUsers:
		return 1
	case stepEnv:
		if m.profileCfg.RequireAdminUser {
			return 2
		}
		return 0
	case stepSites:
		if m.profileCfg.RequireAdminUser {
			return 3
		}
		return 1
	case stepStart:
		if m.profileCfg.RequireAdminUser {
			return 4
		}
		return 2
	case stepReview:
		if m.profileCfg.RequireAdminUser {
			return 5
		}
		return 3
	case stepThresholds:
		if m.profileCfg.RequireAdminUser {
			return 6
		}
		return 4
	case stepDone:
		if m.profileCfg.RequireAdminUser {
			return 7
		}
		return 5
	default:
		return 0
	}
}

func (m model) applianceProgressRail() string {
	labels := m.applianceProgressLabels()
	if labels == nil {
		return ""
	}
	parts := make([]string, len(labels))
	active := m.progressIndex()
	for i, label := range labels {
		if i == active {
			parts[i] = m.theme.TabActive.Render(label)
		} else {
			parts[i] = m.theme.TabIdle.Render(label)
		}
	}
	return strings.Join(parts, " · ")
}
