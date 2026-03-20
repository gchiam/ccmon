// internal/ui/status_bar.go
package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(ColorHeaderFooter)).
			Foreground(Muted)

	warnBarStyle = lipgloss.NewStyle().
			Foreground(UnreadTool)

	hintBarStyle = lipgloss.NewStyle().
			Foreground(Dim)
)

const defaultHint = " ↑↓ nav · Space collapse · Tab switch · Enter select · q quit"

// RenderStatusBar renders the footer bar. warn is non-empty when fsnotify is unavailable.
func RenderStatusBar(warn string, width int) string {
	var content string
	if warn != "" {
		content = warnBarStyle.Render(" ⚠ " + warn)
	} else {
		content = hintBarStyle.Render(defaultHint)
	}
	return statusBarStyle.Width(width).Render(padRight(content, width))
}
