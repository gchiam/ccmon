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

const defaultHint = " ↑↓ nav · Space collapse · Enter select · Tab switch · q quit"

// RenderStatusBar renders the footer bar.
// warn takes priority; info shows full details of the highlighted item; falls back to hint.
func RenderStatusBar(warn, info string, width int) string {
	var content string
	switch {
	case warn != "":
		content = warnBarStyle.Render(" ⚠ " + warn)
	case info != "":
		content = lipgloss.NewStyle().Foreground(Muted).Render(" " + info)
	default:
		content = hintBarStyle.Render(defaultHint)
	}
	return statusBarStyle.Width(width).Render(padRight(content, width))
}
