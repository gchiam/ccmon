// internal/ui/colors.go
package ui

import "github.com/charmbracelet/lipgloss"

// Catppuccin Frappe palette used throughout the TUI.
const (
	ColorBackground      = "#303446"
	ColorPanelBg         = "#292c3c"
	ColorHeaderFooter    = "#232634"
	ColorBorder          = "#414559"
	ColorSelectedRow     = "#414559"
	ColorProjectHeader   = "#ca9ee6"
	ColorActiveSession   = "#a6d189"
	ColorSelectedSession = "#8caaee"
	ColorUnreadTool      = "#e5c890"
	ColorStaleError      = "#e78284"
	ColorMuted           = "#949cbb"
	ColorDim             = "#626880"
)

// Pre-built lipgloss colors for convenience.
var (
	Background    = lipgloss.Color(ColorBackground)
	PanelBg       = lipgloss.Color(ColorPanelBg)
	HeaderFooter  = lipgloss.Color(ColorHeaderFooter)
	Border        = lipgloss.Color(ColorBorder)
	SelectedRow   = lipgloss.Color(ColorSelectedRow)
	ProjectHeader = lipgloss.Color(ColorProjectHeader)
	ActiveSession = lipgloss.Color(ColorActiveSession)
	SelectedSess  = lipgloss.Color(ColorSelectedSession)
	UnreadTool    = lipgloss.Color(ColorUnreadTool)
	StaleError    = lipgloss.Color(ColorStaleError)
	Muted         = lipgloss.Color(ColorMuted)
	Dim           = lipgloss.Color(ColorDim)
)
