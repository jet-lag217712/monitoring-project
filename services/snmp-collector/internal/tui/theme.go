package tui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// ThemeName selects light, dark, or terminal-adaptive colors.
type ThemeName string

const (
	ThemeAuto  ThemeName = "auto"
	ThemeLight ThemeName = "light"
	ThemeDark  ThemeName = "dark"
)

// ParseThemeName maps CLI values to a ThemeName. Unknown values become ThemeAuto.
func ParseThemeName(s string) ThemeName {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "light":
		return ThemeLight
	case "dark":
		return ThemeDark
	default:
		return ThemeAuto
	}
}

// Equate design tokens from frontend/src/index.css (light) and a derived dark palette.
const (
	colorLightBG           = "#f8f7f2"
	colorLightBG2          = "#f6f4ee"
	colorLightInk          = "#0a0a0a"
	colorLightInkSoft      = "#3d3d3d"
	colorLightInkMuted     = "#3d3d3d"
	colorLightBorder       = "#eceae3"
	colorLightBorderStrong = "#7a7a7a"

	colorDarkBG           = "#0a0a0a"
	colorDarkBG2          = "#161514"
	colorDarkInk          = "#f8f7f2"
	colorDarkInkSoft      = "#d4d2cc"
	colorDarkInkMuted     = "#9a978f"
	colorDarkBorder       = "#2a2926"
	colorDarkBorderStrong = "#7a7a7a"

	colorSalmon      = "#e8735a"
	colorSalmonLight = "#fce8e3"
	colorSalmonDark  = "#c75a42"

	colorStatusOK      = "#22c55e"
	colorStatusCaution = "#f59e0b"
	colorStatusAlert   = "#ef4444"
	colorStatusUnknown = "#64748b"

	colorStatusOKText      = "#15803d"
	colorStatusCautionText = "#b45309"
	colorStatusAlertText   = "#b91c1c"
	colorStatusUnknownText = "#475569"
)

// Theme holds resolved lipgloss styles for the operator TUI.
type Theme struct {
	Name ThemeName

	BG           lipgloss.Color
	BG2          lipgloss.Color
	Ink          lipgloss.Color
	InkSoft      lipgloss.Color
	InkMuted     lipgloss.Color
	Border       lipgloss.Color
	BorderStrong lipgloss.Color
	Salmon       lipgloss.Color
	SalmonDark   lipgloss.Color

	StatusOK      lipgloss.Color
	StatusCaution lipgloss.Color
	StatusAlert   lipgloss.Color
	StatusUnknown lipgloss.Color

	LogoMark    lipgloss.Style
	Wordmark    lipgloss.Style
	Eyebrow     lipgloss.Style
	Title       lipgloss.Style
	Muted       lipgloss.Style
	Soft        lipgloss.Style
	Error       lipgloss.Style
	Help        lipgloss.Style
	Panel       lipgloss.Style
	TabActive   lipgloss.Style
	TabIdle     lipgloss.Style
	Label       lipgloss.Style
	Value       lipgloss.Style
	TableHeader lipgloss.Style
	Confirm     lipgloss.Style
	Spinner     lipgloss.Style

	BadgeOK      lipgloss.Style
	BadgeCaution lipgloss.Style
	BadgeAlert   lipgloss.Style
	BadgeUnknown lipgloss.Style
}

// NewTheme builds styles for the given theme preference.
// Honors NO_COLOR by returning a monochrome theme.
func NewTheme(name ThemeName) Theme {
	if os.Getenv("NO_COLOR") != "" {
		return newMonoTheme(name)
	}
	resolved := name
	if name == ThemeAuto {
		if termenv.HasDarkBackground() {
			resolved = ThemeDark
		} else {
			resolved = ThemeLight
		}
	}
	if resolved == ThemeDark {
		return newDarkTheme()
	}
	return newLightTheme()
}

func newLightTheme() Theme {
	t := Theme{
		Name:          ThemeLight,
		BG:            colorLightBG,
		BG2:           colorLightBG2,
		Ink:           colorLightInk,
		InkSoft:       colorLightInkSoft,
		InkMuted:      colorLightInkMuted,
		Border:        colorLightBorder,
		BorderStrong:  colorLightBorderStrong,
		Salmon:        colorSalmon,
		SalmonDark:    colorSalmonDark,
		StatusOK:      colorStatusOK,
		StatusCaution: colorStatusCaution,
		StatusAlert:   colorStatusAlert,
		StatusUnknown: colorStatusUnknown,
	}
	t.buildStyles(colorSalmonLight, colorStatusOKText, colorStatusCautionText, colorStatusAlertText, colorStatusUnknownText)
	return t
}

func newDarkTheme() Theme {
	t := Theme{
		Name:          ThemeDark,
		BG:            colorDarkBG,
		BG2:           colorDarkBG2,
		Ink:           colorDarkInk,
		InkSoft:       colorDarkInkSoft,
		InkMuted:      colorDarkInkMuted,
		Border:        colorDarkBorder,
		BorderStrong:  colorDarkBorderStrong,
		Salmon:        colorSalmon,
		SalmonDark:    colorSalmonDark,
		StatusOK:      colorStatusOK,
		StatusCaution: colorStatusCaution,
		StatusAlert:   colorStatusAlert,
		StatusUnknown: colorStatusUnknown,
	}
	// On dark terminals, badge text uses the bright status colors themselves.
	t.buildStyles("#3a221c", colorStatusOK, colorStatusCaution, colorStatusAlert, colorStatusUnknown)
	return t
}

func newMonoTheme(name ThemeName) Theme {
	t := Theme{Name: name}
	plain := lipgloss.NewStyle()
	t.LogoMark = plain.Bold(true)
	t.Wordmark = plain.Bold(true)
	t.Eyebrow = plain
	t.Title = plain.Bold(true)
	t.Muted = plain.Faint(true)
	t.Soft = plain
	t.Error = plain.Bold(true)
	t.Help = plain.Faint(true)
	t.Panel = plain
	t.TabActive = plain.Bold(true).Underline(true)
	t.TabIdle = plain.Faint(true)
	t.Label = plain.Faint(true)
	t.Value = plain
	t.TableHeader = plain.Bold(true)
	t.Confirm = plain.Bold(true)
	t.Spinner = plain
	t.BadgeOK = plain
	t.BadgeCaution = plain
	t.BadgeAlert = plain
	t.BadgeUnknown = plain
	return t
}

func (t *Theme) buildStyles(salmonBg, okText, cautionText, alertText, unknownText string) {
	t.LogoMark = lipgloss.NewStyle().Foreground(t.Salmon).Bold(true)
	t.Wordmark = lipgloss.NewStyle().Foreground(t.Ink).Bold(true)
	t.Eyebrow = lipgloss.NewStyle().Foreground(t.Salmon).Bold(true)
	t.Title = lipgloss.NewStyle().Foreground(t.Ink).Bold(true)
	t.Muted = lipgloss.NewStyle().Foreground(t.InkMuted)
	t.Soft = lipgloss.NewStyle().Foreground(t.InkSoft)
	t.Error = lipgloss.NewStyle().Foreground(t.StatusAlert).Bold(true)
	t.Help = lipgloss.NewStyle().Foreground(t.InkMuted)
	t.Panel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(0, 1)
	t.TabActive = lipgloss.NewStyle().
		Foreground(t.Salmon).
		Bold(true).
		Underline(true)
	t.TabIdle = lipgloss.NewStyle().Foreground(t.InkMuted)
	t.Label = lipgloss.NewStyle().Foreground(t.InkMuted).Bold(true)
	t.Value = lipgloss.NewStyle().Foreground(t.Ink)
	t.TableHeader = lipgloss.NewStyle().Foreground(t.InkMuted).Bold(true)
	t.Confirm = lipgloss.NewStyle().Foreground(t.Salmon).Bold(true)
	t.Spinner = lipgloss.NewStyle().Foreground(t.Salmon)

	t.BadgeOK = lipgloss.NewStyle().Foreground(lipgloss.Color(okText)).Bold(true)
	t.BadgeCaution = lipgloss.NewStyle().Foreground(lipgloss.Color(cautionText)).Bold(true)
	t.BadgeAlert = lipgloss.NewStyle().Foreground(lipgloss.Color(alertText)).Bold(true)
	t.BadgeUnknown = lipgloss.NewStyle().Foreground(lipgloss.Color(unknownText)).Bold(true)
	_ = salmonBg // reserved for future selected-row backgrounds
}

// StatusStyle returns the badge style for a health state string.
func (t Theme) StatusStyle(state string) lipgloss.Style {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "ok", "healthy", "normal", "up":
		return t.BadgeOK
	case "warning", "caution", "degraded":
		return t.BadgeCaution
	case "critical", "alert", "down", "failed", "error":
		return t.BadgeAlert
	case "unknown", "upstream_unreachable", "dependency_impacted":
		return t.BadgeUnknown
	default:
		if state == "" {
			return t.BadgeUnknown
		}
		return t.Soft
	}
}
