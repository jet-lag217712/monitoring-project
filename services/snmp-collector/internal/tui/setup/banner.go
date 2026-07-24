package setup

import (
	"github.com/equate/ogsd/services/snmp-collector/internal/tui"
)

func withTopPadding(content string) string {
	return tui.WithTopPadding(content)
}

func withSectionGap(content string) string {
	return tui.WithSectionGap(content)
}

func splashLogo(th tui.Theme) string {
	return tui.SplashLogo(th)
}

func configuratorLogo(th tui.Theme) string {
	return tui.ConfiguratorLogo(th)
}
