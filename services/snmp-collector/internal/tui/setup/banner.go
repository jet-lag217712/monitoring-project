package setup

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/equate/ogsd/services/snmp-collector/internal/tui"
)

// Block art generated with:
//   npx oh-my-logo "<text>" --filled --block-font block --no-color

const artSlashes = `     ██╗     ██╗
    ██╔╝    ██╔╝
   ██╔╝    ██╔╝
  ██╔╝    ██╔╝
 ██╔╝    ██╔╝
 ╚═╝     ╚═╝`

const artEquate = `███████╗  ██████╗  ██╗   ██╗  █████╗  ████████╗ ███████╗
██╔════╝ ██╔═══██╗ ██║   ██║ ██╔══██╗ ╚══██╔══╝ ██╔════╝
█████╗   ██║   ██║ ██║   ██║ ███████║    ██║    █████╗
██╔══╝   ██║▄▄ ██║ ██║   ██║ ██╔══██║    ██║    ██╔══╝
███████╗ ╚██████╔╝ ╚██████╔╝ ██║  ██║    ██║    ███████╗
╚══════╝  ╚══▀▀═╝   ╚═════╝  ╚═╝  ╚═╝    ╚═╝    ╚══════╝`

const artCfg = `      ██████╗ ███████╗  ██████╗
     ██╔════╝ ██╔════╝ ██╔════╝
     ██║      █████╗   ██║  ███╗
     ██║      ██╔══╝   ██║   ██║
 ██╗ ╚██████╗ ██║      ╚██████╔╝
 ╚═╝  ╚═════╝ ╚═╝       ╚═════╝`

const artTechno = ` ████████╗ ███████╗  ██████╗ ██╗  ██╗ ███╗   ██╗  ██████╗
 ╚══██╔══╝ ██╔════╝ ██╔════╝ ██║  ██║ ████╗  ██║ ██╔═══██╗
    ██║    █████╗   ██║      ███████║ ██╔██╗ ██║ ██║   ██║
    ██║    ██╔══╝   ██║      ██╔══██║ ██║╚██╗██║ ██║   ██║
    ██║    ███████╗ ╚██████╗ ██║  ██║ ██║ ╚████║ ╚██████╔╝
    ╚═╝    ╚══════╝  ╚═════╝ ╚═╝  ╚═╝ ╚═╝  ╚═══╝  ╚═════╝`

const artLogies = ` ██╗       ██████╗   ██████╗  ██╗ ███████╗ ███████╗
 ██║      ██╔═══██╗ ██╔════╝  ██║ ██╔════╝ ██╔════╝
 ██║      ██║   ██║ ██║  ███╗ ██║ █████╗   ███████╗
 ██║      ██║   ██║ ██║   ██║ ██║ ██╔══╝   ╚════██║
 ███████╗ ╚██████╔╝ ╚██████╔╝ ██║ ███████╗ ███████║
 ╚══════╝  ╚═════╝   ╚═════╝  ╚═╝ ╚══════╝ ╚══════╝`

const viewTopPadding = 2

// sectionGapLines targets ~60-70px between the logo block and the next section,
// assuming a typical terminal cell height of ~16-18px.
const sectionGapLines = 4

func withTopPadding(content string) string {
	return lipgloss.NewStyle().PaddingTop(viewTopPadding).Render(content)
}

func withSectionGap(content string) string {
	return lipgloss.NewStyle().PaddingTop(sectionGapLines).Render(content)
}

func joinHorizontal(parts ...string) string {
	styled := make([]string, len(parts))
	for i, p := range parts {
		styled[i] = p
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, styled...)
}

func splashLogo(th tui.Theme) string {
	slashes := styleBlockArt(artSlashes, th.Salmon, th.SalmonDark)
	equate := styleBlockArt(artEquate, th.Ink, th.InkSoft)
	cfg := styleBlockArt(artCfg, th.Ink, th.InkSoft)
	return joinHorizontal(slashes, "  ", equate, cfg)
}

func configuratorLogo(th tui.Theme) string {
	equate := joinHorizontal(" ", styleBlockArt(artEquate, th.Salmon, th.SalmonDark))
	techno := styleBlockArt(artTechno, th.Salmon, th.SalmonDark)
	logies := styleBlockArt(artLogies, th.Salmon, th.SalmonDark)
	technologies := joinHorizontal(techno, logies)
	return lipgloss.JoinVertical(lipgloss.Left, equate, technologies)
}

func styleBlockArt(art string, face, shadow lipgloss.Color) string {
	faceStyle := lipgloss.NewStyle().Foreground(face)
	shadowStyle := lipgloss.NewStyle().Foreground(shadow)
	lines := strings.Split(strings.TrimRight(art, "\n"), "\n")
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = styleBlockLine(line, faceStyle, shadowStyle)
	}
	return strings.Join(out, "\n")
}

func styleBlockLine(line string, face, shadow lipgloss.Style) string {
	var b strings.Builder
	for _, r := range line {
		switch {
		case r == '█':
			b.WriteString(face.Render(string(r)))
		case isBlockShadowRune(r):
			b.WriteString(shadow.Render(string(r)))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isBlockShadowRune(r rune) bool {
	if unicode.IsSpace(r) {
		return false
	}
	return r != '█'
}
