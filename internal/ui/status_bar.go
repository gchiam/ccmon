// internal/ui/status_bar.go
package ui

import (
	"strings"

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

const defaultHint = "↑↓ nav · Space collapse · Enter select · Tab switch · q quit "

// RenderStatusBar renders the footer bar.
// warn is shown on the left when fsnotify is unavailable; info shows full details of
// the highlighted item on the left. Keyboard hints are always right-aligned.
func RenderStatusBar(warn, info string, width int) string {
	hint := hintBarStyle.Render(defaultHint)
	hintW := lipgloss.Width(hint)

	var left string
	if warn != "" {
		left = warnBarStyle.Render(" ⚠ " + warn)
	} else if info != "" {
		left = lipgloss.NewStyle().Foreground(Muted).Render(" " + info)
	}

	leftW := lipgloss.Width(left)
	gap := width - leftW - hintW
	if gap < 1 {
		gap = 1
	}
	content := left + strings.Repeat(" ", gap) + hint
	return statusBarStyle.Width(width).Render(content)
}
