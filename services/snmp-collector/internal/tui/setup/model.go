package setup

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	stepAdminUser
	stepUsers
)

type model struct {
	deployDir string
	theme     tui.Theme
	version   string
	profile   Profile
	profileCfg ProfileConfig
	splash    bool
	width     int
	height    int
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

	adminUsernameInput textinput.Model
	adminPasswordInput textinput.Model
	adminConfirmInput  textinput.Model
	adminFocus         int

	usersBody      string
	usersMode      string
	usersUsername  textinput.Model
	usersPassword  textinput.Model
	usersConfirm   textinput.Model
	usersFocus     int

	reviewSiteIdx    int
	reviewCandidates []map[string]any
	reviewApproved   []bool
	reviewCursor     int
	reviewScrollTop  int
	reviewPhase      string
	reviewSiteLines  []string

	confirmQuit bool

	adminPhase string
}

func styleTextInput(ti textinput.Model, th tui.Theme) textinput.Model {
	ti.Prompt = ""
	ti.TextStyle = lipgloss.NewStyle().Foreground(th.Ink)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(th.InkMuted)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(th.Ink).Background(th.Ink)
	return ti
}

func newModel(deployDir string, theme tui.Theme, version string, profile Profile) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = theme.Spinner

	cfg := ProfileConfigFor(profile)

	envInputs := newEnvInputs(theme, profile)

	siteCountInput := styleTextInput(textinput.New(), theme)
	siteCountInput.Placeholder = "number of site containers"
	siteCountInput.CharLimit = 2
	siteCountInput.Width = 8
	siteCountInput.SetValue(strconv.Itoa(defaultSiteCount))

	thresholdInput := styleTextInput(textinput.New(), theme)
	thresholdInput.Placeholder = "temperature warning °C"
	thresholdInput.CharLimit = 4
	thresholdInput.Width = 8
	thresholdInput.SetValue("65")

	adminUsernameInput := styleTextInput(textinput.New(), theme)
	adminUsernameInput.Placeholder = "admin username"
	adminUsernameInput.CharLimit = 32
	adminUsernameInput.Width = 24

	adminPasswordInput := styleTextInput(textinput.New(), theme)
	adminPasswordInput.Placeholder = "password (required)"
	adminPasswordInput.CharLimit = 128
	adminPasswordInput.Width = 32
	adminPasswordInput.EchoMode = textinput.EchoPassword

	adminConfirmInput := styleTextInput(textinput.New(), theme)
	adminConfirmInput.Placeholder = "confirm password"
	adminConfirmInput.CharLimit = 128
	adminConfirmInput.Width = 32
	adminConfirmInput.EchoMode = textinput.EchoPassword

	usersUsername := styleTextInput(textinput.New(), theme)
	usersUsername.Placeholder = "username"
	usersUsername.CharLimit = 32
	usersUsername.Width = 24

	usersPassword := styleTextInput(textinput.New(), theme)
	usersPassword.Placeholder = "password"
	usersPassword.CharLimit = 128
	usersPassword.Width = 32
	usersPassword.EchoMode = textinput.EchoPassword

	usersConfirm := styleTextInput(textinput.New(), theme)
	usersConfirm.Placeholder = "confirm password"
	usersConfirm.CharLimit = 128
	usersConfirm.Width = 32
	usersConfirm.EchoMode = textinput.EchoPassword

	m := model{
		deployDir:          deployDir,
		theme:              theme,
		version:            version,
		profile:            profile,
		profileCfg:         cfg,
		splash:             true,
		step:               stepEnv,
		spinner:            sp,
		envInputs:          envInputs,
		siteCountInput:     siteCountInput,
		thresholdInput:     thresholdInput,
		probeRate:          "5",
		probeBurst:         "2",
		adminUsernameInput: adminUsernameInput,
		adminPasswordInput: adminPasswordInput,
		adminConfirmInput:  adminConfirmInput,
		usersUsername:      usersUsername,
		usersPassword:      usersPassword,
		usersConfirm:       usersConfirm,
		usersMode:          "menu",
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
		siteIDs[i] = styleTextInput(textinput.New(), m.theme)
		siteIDs[i].Placeholder = siteIDForIndex(i + 1)
		siteIDs[i].CharLimit = maxSiteIDLen
		siteIDs[i].Width = 20
		if i < len(m.siteIDInputs) && strings.TrimSpace(m.siteIDInputs[i].Value()) != "" {
			siteIDs[i].SetValue(m.siteIDInputs[i].Value())
		} else {
			siteIDs[i].SetValue(siteIDForIndex(i + 1))
		}

		cidrs[i] = styleTextInput(textinput.New(), m.theme)
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
	sites []SiteSpec
}

type existingAdminsMsg struct {
	err         error
	body        string
	hasExisting bool
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		if next, cmd, handled := m.handleQuitKey(msg); handled {
			return next, cmd
		}
		if m.splash {
			if msg.String() == "enter" {
				m.splash = false
				m.step = m.firstStepAfterSplash()
				if m.step == stepAdminUser {
					m.adminFocus = 0
					m = m.updateAdminFocus()
					if m.profile == ProfileAppliance {
						m.adminPhase = "loading"
						m.loading = true
						return m, m.loadExistingAdmins()
					}
					m.adminPhase = "create"
				} else if m.step == stepEnv {
					m.envFocus = 0
					m = m.updateEnvFocus()
				}
				return m, textinput.Blink
			}
			return m, nil
		}
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
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.loading {
			return m, cmd
		}
	case reviewScanMsg:
		m.loading = false
		m.err = ""
		m.body = msg.body
		m.reviewCandidates = msg.candidates
		m.reviewApproved = msg.approved
		m.reviewCursor = 0
		m.reviewScrollTop = 0
		m.reviewPhase = "pick"
		return m, nil
	case reviewAcceptMsg:
		m.loading = false
		m.err = ""
		m.reviewSiteLines = append(m.reviewSiteLines, msg.line)
		if m.reviewSiteIdx+1 >= len(m.sites) {
			m.step = stepThresholds
			m.body = strings.Join(m.reviewSiteLines, "\n")
		} else {
			m.reviewSiteIdx++
			m.reviewPhase = ""
			m.reviewCandidates = nil
			m.reviewApproved = nil
			m.reviewCursor = 0
			m.reviewScrollTop = 0
			m.body = ""
		}
		return m, nil
	case existingAdminsMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.adminPhase = "create"
			return m, nil
		}
		m.err = ""
		if msg.hasExisting {
			m.adminPhase = "choose"
			m.body = msg.body
			return m, nil
		}
		m.adminPhase = "create"
		m.body = ""
		return m, textinput.Blink
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
		case stepAdminUser:
			if m.profileCfg.UserManagement {
				m.step = stepUsers
				m.usersMode = "menu"
				return m, m.refreshUsersList()
			}
			m.step = stepEnv
			m.envFocus = 0
			m = m.updateEnvFocus()
			return m, textinput.Blink
		case stepUsers:
			if m.usersMode != "menu" {
				m.usersMode = "menu"
				m.usersUsername.SetValue("")
				m.usersPassword.SetValue("")
				m.usersConfirm.SetValue("")
			}
			if strings.TrimSpace(msg.body) != "" {
				m.usersBody = msg.body
			}
			return m, nil
		case stepSites:
			if len(msg.sites) > 0 {
				m.sites = msg.sites
			}
			m.step = stepStart
			return m, nil
		case stepStart:
			m.step = stepReview
			if !m.profileCfg.AutoAcceptDiscovery {
				m.reviewSiteIdx = 0
				m.reviewSiteLines = nil
				m.reviewPhase = ""
				m.body = ""
			}
			return m, nil
		case stepReview:
			if m.profileCfg.AutoAcceptDiscovery {
				m.step = stepThresholds
			}
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
	case stepAdminUser:
		return m.updateAdminUser(msg)
	case stepUsers:
		return m.updateUsers(msg)
	case stepEnv:
		return m.updateEnv(msg)
	case stepSites:
		return m.updateSites(msg)
	case stepStart:
		return m.updateStart(msg)
	case stepReview:
		if m.profileCfg.AutoAcceptDiscovery {
			if m.err == "" {
				return m.updateReview(msg)
			}
			return m, nil
		}
		return m.updateReviewManual(msg)
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
	profile := m.profile
	values, err := m.sharedEnvValues()
	if err != nil {
		return func() tea.Msg { return asyncDoneMsg{err: err} }
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
		specs, err := BuildSiteSpecs(profile, count, siteIDs, cidrs)
		if err != nil {
			return asyncDoneMsg{err: err}
		}
		if err := writeEnvFile(envPath(deployDir), values); err != nil {
			return asyncDoneMsg{err: err}
		}
		applyEnvToProcess(values)
		if err := persistMultiSiteArtifacts(deployDir, profile, specs, rate, burst); err != nil {
			return asyncDoneMsg{err: err}
		}
		body := fmt.Sprintf("Saved shared .env, manifest, and %d site artifact(s).", len(specs))
		return asyncDoneMsg{body: body, sites: specs}
	}
}

func (m model) runStart() tea.Cmd {
	deployDir := m.deployDir
	profile := m.profile
	return func() tea.Msg {
		manifest, err := LoadManifest(deployDir)
		if err != nil {
			return asyncDoneMsg{err: err}
		}
		if err := startSiteCollectors(deployDir, profile, manifest.Sites); err != nil {
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
			line, err := reviewSiteAuto(spec, deployDir)
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
	if m.splash {
		return m.viewSplash()
	}
	th := m.theme
	var body strings.Builder
	if rail := m.applianceProgressRail(); rail != "" {
		body.WriteString(rail)
	} else {
		body.WriteString(m.progressRail())
	}
	body.WriteString("\n\n")
	if m.err != "" {
		body.WriteString(th.Error.Render("error: " + m.err))
		body.WriteString("\n\n")
	}
	if m.loading {
		body.WriteString(th.Spinner.Render(m.spinner.View()))
		body.WriteString(" ")
		switch m.step {
		case stepAdminUser:
			body.WriteString(th.Muted.Render("Checking administrator accounts…"))
		case stepStart:
			if m.profile == ProfileAppliance {
				body.WriteString(th.Muted.Render("Starting all site collectors…"))
			} else {
				body.WriteString(th.Muted.Render("Building image and waiting for all site collectors…"))
			}
		case stepReview:
			if m.profileCfg.AutoAcceptDiscovery {
				body.WriteString(th.Muted.Render("Scanning each site CIDR and accepting devices…"))
			} else {
				body.WriteString(th.Muted.Render("Scanning site for discovery candidates…"))
			}
		case stepThresholds:
			body.WriteString(th.Muted.Render("Applying threshold to all sites…"))
		default:
			body.WriteString(th.Muted.Render("working…"))
		}
		m.appendQuitFooter(th, &body)
		return withTopPadding(lipgloss.JoinVertical(lipgloss.Left, configuratorLogo(th), withSectionGap(body.String())))
	}
	switch m.step {
	case stepAdminUser:
		body.WriteString(m.viewAdminUser(th))
	case stepUsers:
		body.WriteString(m.viewUsers(th))
	case stepEnv:
		if m.profile == ProfileAppliance {
			body.WriteString(m.viewApplianceEnv(th))
		} else {
			body.WriteString(th.Title.Render("Step 1 - Shared environment"))
			body.WriteString("\n\n")
			for i, input := range m.envInputs {
				body.WriteString(th.Soft.Render(fmt.Sprintf("%d", i+1)))
				body.WriteString(" ")
				body.WriteString(th.Confirm.Render(">"))
				body.WriteString(" ")
				body.WriteString(input.View())
				body.WriteString("\n")
			}
			body.WriteString(th.Muted.Render("tab next field · enter continue"))
		}
	case stepSites:
		body.WriteString(th.Title.Render("Step 2 - Site containers"))
		body.WriteString("\n\n")
		body.WriteString(th.Label.Render("count"))
		body.WriteString(" ")
		body.WriteString(m.siteCountInput.View())
		body.WriteString("\n")
		for i := range m.cidrInputs {
			body.WriteString(th.Label.Render("site id"))
			body.WriteString(" ")
			body.WriteString(m.siteIDInputs[i].View())
			body.WriteString("  ")
			body.WriteString(th.Label.Render("cidr"))
			body.WriteString(" ")
			body.WriteString(m.cidrInputs[i].View())
			body.WriteString("\n")
		}
		if m.body != "" {
			body.WriteString("\n")
			body.WriteString(th.Value.Render(m.body))
			body.WriteString("\n")
		}
		body.WriteString("\n")
		body.WriteString(th.Muted.Render("tab next field · enter on last CIDR to save artifacts"))
	case stepStart, stepReview:
		if m.step == stepAdminUser && m.loading {
			body.WriteString(th.Muted.Render("Checking administrator accounts…"))
		} else if m.step == stepStart {
			body.WriteString(th.Title.Render("Step 3 - Starting collectors"))
			body.WriteString("\n\n")
			if m.body != "" {
				body.WriteString(th.Value.Render(m.body))
				body.WriteString("\n\n")
			}
			if m.profile == ProfileAppliance {
				body.WriteString(th.Muted.Render("enter to start all site containers"))
			} else {
				body.WriteString(th.Muted.Render("enter to build and start all site containers"))
			}
		} else {
			if m.profileCfg.AutoAcceptDiscovery {
				body.WriteString(th.Title.Render("Step 4 - Review inventory"))
				body.WriteString("\n")
				if m.body != "" {
					body.WriteString(th.Value.Render(m.body))
					body.WriteString("\n")
				}
				if m.err != "" {
					body.WriteString("\n")
					body.WriteString(th.Muted.Render("r retry discovery · s skip and continue"))
				} else if !m.loading {
					body.WriteString("\n")
					body.WriteString(th.Muted.Render("enter to run discovery for each site"))
				}
			} else {
				body.WriteString(m.viewReviewManual(th))
			}
		}
	case stepThresholds:
		body.WriteString(th.Title.Render("Step 5 - Thresholds"))
		body.WriteString("\n\n")
		body.WriteString(th.Label.Render("Global temperature warning °C"))
		body.WriteString(" ")
		body.WriteString(m.thresholdInput.View())
		body.WriteString("\n\n")
		body.WriteString(th.Muted.Render("enter to apply to all sites and finish setup"))
	case stepDone:
		body.WriteString(th.Title.Render("Setup complete"))
		body.WriteString("\n\n")
		body.WriteString(th.Value.Render(m.body))
		body.WriteString("\n\n")
		body.WriteString(th.Muted.Render("Per-site operator TUI examples:"))
		body.WriteString("\n")
		if manifest, err := LoadManifest(m.deployDir); err == nil {
			for _, spec := range manifest.Sites {
				body.WriteString(th.Muted.Render(fmt.Sprintf(
					"docker compose exec -it %s /collector tui -socket /run/snmp-collector/control.sock -theme auto",
					spec.ServiceName,
				)))
				body.WriteString("\n")
			}
		}
		body.WriteString(th.Muted.Render("press q or enter to quit"))
	}
	m.appendQuitFooter(th, &body)
	return withTopPadding(lipgloss.JoinVertical(lipgloss.Left, configuratorLogo(th), withSectionGap(body.String())))
}

func (m model) progressRail() string {
	labels := []string{"Env", "Sites", "Start", "Review", "Thresholds", "Done"}
	parts := make([]string, len(labels))
	for i, label := range labels {
		if step(i) == m.step {
			parts[i] = m.theme.TabActive.Render(label)
		} else {
			parts[i] = m.theme.TabIdle.Render(label)
		}
	}
	return strings.Join(parts, " · ")
}

func (m model) viewSplash() string {
	th := m.theme
	width := m.width
	height := m.height
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}

	var belowLogo strings.Builder
	belowLogo.WriteString(th.Title.Render("Equate SNMP Collector Configuration"))
	belowLogo.WriteString("\n\n")
	belowLogo.WriteString(th.Confirm.Render("→"))
	belowLogo.WriteString(th.Muted.Render(" enter to continue · ctrl+c to quit"))

	headerBlock := lipgloss.NewStyle().Padding(1, 0, 0, 0).Render(
		lipgloss.JoinVertical(lipgloss.Left, splashLogo(th), withSectionGap(belowLogo.String())),
	)

	ver := formatVersion(m.version)
	heart := lipgloss.NewStyle().Foreground(th.StatusAlert).Render("❤")
	footerMuted := lipgloss.NewStyle().Foreground(th.InkMuted).Faint(true)
	footerLine := footerMuted.Render("V."+ver+" · Made with ") + heart + footerMuted.Render(" in West Lafayette")
	footer := lipgloss.Place(width, 1, lipgloss.Right, lipgloss.Center, footerLine)

	bodyHeight := max(1, height-lipgloss.Height(footer)-viewTopPadding)
	body := lipgloss.Place(width, bodyHeight, lipgloss.Left, lipgloss.Top, headerBlock)
	return withTopPadding(lipgloss.JoinVertical(lipgloss.Left, body, footer))
}

func formatVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" || version == "unknown" {
		return "1.5.0"
	}
	return version
}
