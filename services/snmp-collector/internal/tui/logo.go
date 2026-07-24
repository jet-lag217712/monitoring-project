package tui

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
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

const ViewTopPadding = 2

// SectionGapLines targets ~60-70px between the logo block and the next section,
// assuming a typical terminal cell height of ~16-18px.
const SectionGapLines = 4

// Logo returns the runtime TUI block-art banner: //EQUATE.
func Logo(th Theme) string {
	slashes := StyleBlockArt(artSlashes, th.Salmon, th.SalmonDark)
	equate := StyleBlockArt(artEquate, th.Ink, th.InkSoft)
	return JoinBlocks(slashes, "  ", equate)
}

// SplashLogo returns the setup splash banner: //EQUATECFG.
func SplashLogo(th Theme) string {
	slashes := StyleBlockArt(artSlashes, th.Salmon, th.SalmonDark)
	equate := StyleBlockArt(artEquate, th.Ink, th.InkSoft)
	cfg := StyleBlockArt(artCfg, th.Ink, th.InkSoft)
	return JoinBlocks(slashes, "  ", equate, cfg)
}

// ConfiguratorLogo returns the setup configurator banner.
func ConfiguratorLogo(th Theme) string {
	equate := JoinBlocks(" ", StyleBlockArt(artEquate, th.Salmon, th.SalmonDark))
	techno := StyleBlockArt(artTechno, th.Salmon, th.SalmonDark)
	logies := StyleBlockArt(artLogies, th.Salmon, th.SalmonDark)
	technologies := JoinBlocks(techno, logies)
	return lipgloss.JoinVertical(lipgloss.Left, equate, technologies)
}

// WithTopPadding adds standard top padding to content.
func WithTopPadding(content string) string {
	return lipgloss.NewStyle().PaddingTop(ViewTopPadding).Render(content)
}

// WithSectionGap adds spacing between the logo block and the next section.
func WithSectionGap(content string) string {
	return lipgloss.NewStyle().PaddingTop(SectionGapLines).Render(content)
}

// JoinBlocks joins styled blocks horizontally, top-aligned.
func JoinBlocks(parts ...string) string {
	styled := make([]string, len(parts))
	for i, p := range parts {
		styled[i] = p
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, styled...)
}

// StyleBlockArt colors block-font art with a face and shadow palette.
func StyleBlockArt(art string, face, shadow lipgloss.Color) string {
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
