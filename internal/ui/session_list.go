// internal/ui/session_list.go
package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/gchiam/ccmon/internal/session"
)

var (
	sessionListStyle = lipgloss.NewStyle().
				Background(PanelBg).
				Foreground(lipgloss.Color("#c6d0f5"))

	projectHeaderStyle = lipgloss.NewStyle().
				Foreground(ProjectHeader).
				Bold(true)

	selectedRowStyle = lipgloss.NewStyle().
				Background(SelectedRow).
				Foreground(SelectedSess)

	staleStyle = lipgloss.NewStyle().
			Foreground(StaleError).
			Faint(true)

	mutedStyle = lipgloss.NewStyle().Foreground(Muted)
	dimStyle   = lipgloss.NewStyle().Foreground(Dim)
)

// SessionListState holds cursor and collapse state for the left panel.
type SessionListState struct {
	Cursor       int             // flat index into visible rows
	Collapsed    map[string]bool // project name -> collapsed
	UnreadSet    map[string]bool // sessionID -> has unread
	FocusedPanel int             // 0=left, 1=right
}

// RenderSessionList renders the left panel.
func RenderSessionList(groups []session.ProjectGroup, selected string, state SessionListState, width, height int) string {
	var lines []string

	flatIdx := 0
	for _, g := range groups {
		collapsed := state.Collapsed[g.Name]
		arrow := "v"
		if collapsed {
			arrow = ">"
		}
		groupText := truncateLine(fmt.Sprintf(" %s %s", arrow, g.Name), width)
		var groupLine string
		if flatIdx == state.Cursor {
			groupLine = selectedRowStyle.Render(padRight(groupText, width))
		} else {
			groupLine = projectHeaderStyle.Render(groupText)
		}
		lines = append(lines, padRight(groupLine, width))
		flatIdx++

		if !collapsed {
			for _, s := range g.Sessions {
				row := truncateLine(renderSessionRow(s, selected, state), width)
				if flatIdx == state.Cursor {
					row = selectedRowStyle.Render(padRight(row, width))
				}
				lines = append(lines, padRight(row, width))
				flatIdx++
			}
		}
	}

	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return sessionListStyle.Width(width).Render(strings.Join(lines[:minInt(len(lines), height)], "\n"))
}

func renderSessionRow(s *session.Session, selected string, state SessionListState) string {
	prefix := "   "
	unread := state.UnreadSet[s.SessionID]

	var dot string
	switch {
	case s.SessionID == selected:
		dot = lipgloss.NewStyle().Foreground(SelectedSess).Render("●")
	case !s.Alive:
		dot = lipgloss.NewStyle().Foreground(StaleError).Render("⚠")
	case unread:
		dot = lipgloss.NewStyle().Foreground(UnreadTool).Render("●")
	default:
		dot = lipgloss.NewStyle().Foreground(ActiveSession).Render("●")
	}

	name := s.ProjectName
	if s.WorktreeBranch != "" {
		name = "⎇ " + s.WorktreeBranch
	}
	if unread {
		name = name + " ✦"
	}

	if !s.Alive {
		return staleStyle.Render(fmt.Sprintf("%s%s %-14s process dead", prefix, dot, name))
	}
	ts := formatStart(s.StartedAt)
	return fmt.Sprintf("%s%s %-14s %s", prefix, dot, name, dimStyle.Render(ts))
}

// stripANSI removes SGR color escape sequences (ESC[...m) from s so it can be re-styled cleanly.
// It only handles SGR codes (ending in 'm'), which is what lipgloss emits.
func stripANSI(s string) string {
	var out strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			inEsc = true
		case inEsc && r == 'm':
			inEsc = false
		case inEsc:
			// still inside escape sequence
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}

func formatStart(unix int64) string {
	if unix == 0 {
		return ""
	}
	return time.Unix(unix, 0).Format("15:04:05")
}

// CountSessions returns the total number of sessions across all groups.
// Exported for testing.
func CountSessions(groups []session.ProjectGroup) int {
	n := 0
	for _, g := range groups {
		n += len(g.Sessions)
	}
	return n
}

func padRight(s string, width int) string {
	visible := lipgloss.Width(s)
	if visible < width {
		return s + strings.Repeat(" ", width-visible)
	}
	return s
}

// truncateLine clips s to at most maxW visible characters, preserving ANSI codes
// by stripping them first and then re-rendering the plain truncated text.
func truncateLine(s string, maxW int) string {
	plain := stripANSI(s)
	runes := []rune(plain)
	if len(runes) > maxW {
		runes = runes[:maxW]
	}
	return string(runes)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
