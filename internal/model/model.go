// internal/model/model.go
package model

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gchiam/ccmon/internal/session"
	"github.com/gchiam/ccmon/internal/ui"
)

// Model is the Bubble Tea model.
type Model struct {
	width  int
	height int

	groups     []session.ProjectGroup
	selectedID string

	listState  ui.SessionListState
	convScroll int
	focusRight bool

	entries   []ui.ConversationEntry
	unreadSet map[string]bool

	watchWarn string
	fatalMsg  string

	// onSelectSession is called (from main.go) when the user selects a session.
	// main.go is responsible for reader goroutine lifecycle.
	onSelectSession func(*session.Session)
}

// New creates a fresh Model.
func New() *Model {
	return &Model{
		listState: ui.SessionListState{
			Collapsed: make(map[string]bool),
			UnreadSet: make(map[string]bool),
		},
		unreadSet: make(map[string]bool),
	}
}

// OnSelectSession registers a callback invoked when the user selects a new session.
// main.go uses this to start/stop the reader goroutine.
func (m *Model) OnSelectSession(fn func(*session.Session)) {
	m.onSelectSession = fn
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		return m.handleKey(msg)

	case SessionsUpdatedMsg:
		m.groups = groupAndMerge(msg.Sessions, m.groups)
		m.listState.UnreadSet = m.unreadSet

	case UnreadMsg:
		if msg.SessionID != m.selectedID {
			m.unreadSet[msg.SessionID] = true
			m.listState.UnreadSet = m.unreadSet
		}

	case ConversationMsg:
		if msg.SessionID == m.selectedID {
			m.entries = append(m.entries, msg.Entries...)
			m.convScroll = 1<<31 - 1 // stay pinned to bottom; clamped on render
		}

	case WatchErrorMsg:
		m.watchWarn = "File watching unavailable, polling every 2s"
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "tab":
		m.focusRight = !m.focusRight
		if m.focusRight {
			m.listState.FocusedPanel = 1
		} else {
			m.listState.FocusedPanel = 0
		}

	case "up":
		if !m.focusRight {
			if m.listState.Cursor > 0 {
				m.listState.Cursor--
			}
		} else if m.convScroll > 0 {
			m.convScroll--
		}

	case "down":
		if !m.focusRight {
			maxIdx := flatCount(m.groups, m.listState.Collapsed) - 1
			if m.listState.Cursor < maxIdx {
				m.listState.Cursor++
			}
		} else {
			m.convScroll++
		}

	case " ":
		if !m.focusRight {
			m.toggleCollapse()
		}

	case "enter":
		if !m.focusRight {
			m.selectCurrent()
			m.focusRight = true
			m.listState.FocusedPanel = 1
		}
	}
	return m, nil
}

func (m *Model) toggleCollapse() {
	idx := 0
	for i, g := range m.groups {
		if idx == m.listState.Cursor {
			m.groups[i].Collapsed = !g.Collapsed
			m.listState.Collapsed[g.Name] = m.groups[i].Collapsed
			return
		}
		idx++
		if !m.listState.Collapsed[g.Name] {
			idx += len(g.Sessions)
		}
	}
}

func (m *Model) selectCurrent() {
	idx := 0
	for _, g := range m.groups {
		if idx == m.listState.Cursor {
			return // cursor on group header
		}
		idx++
		if !m.listState.Collapsed[g.Name] {
			for _, s := range g.Sessions {
				if idx == m.listState.Cursor {
					m.selectedID = s.SessionID
					delete(m.unreadSet, s.SessionID)
					m.listState.UnreadSet = m.unreadSet
					m.entries = nil
					m.convScroll = 1<<31 - 1 // clamped to maxScroll on render
					if m.onSelectSession != nil {
						m.onSelectSession(s)
					}
					return
				}
				idx++
			}
		}
	}
}

func (m *Model) View() string {
	if m.fatalMsg != "" {
		return renderFatal(m.fatalMsg, m.width, m.height)
	}
	if m.width < 80 || m.height < 24 {
		return renderFatal("⚠ Terminal too small, resize to at least 80x24 to continue", m.width, m.height)
	}

	leftWidth := 24
	rightWidth := m.width - leftWidth - 1
	contentHeight := m.height - 2

	header := renderHeader(m.selectedID, m.groups, leftWidth, rightWidth)
	leftPanel := ui.RenderSessionList(m.groups, m.selectedID, m.listState, leftWidth, contentHeight)
	rightPanel := ui.RenderConversation(m.entries, m.convScroll, rightWidth, contentHeight)
	divider := renderDivider(contentHeight)
	footer := ui.RenderStatusBar(m.watchWarn, m.width)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, divider, rightPanel)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func renderHeader(selectedID string, groups []session.ProjectGroup, leftW, rightW int) string {
	left := lipgloss.NewStyle().
		Background(lipgloss.Color(ui.ColorHeaderFooter)).
		Foreground(lipgloss.Color(ui.ColorProjectHeader)).
		Width(leftW).
		Render(" Sessions")

	projName, worktree := "", ""
	for _, g := range groups {
		for _, s := range g.Sessions {
			if s.SessionID == selectedID {
				projName = s.ProjectName
				worktree = s.WorktreeBranch
			}
		}
	}
	title := " Conversation"
	if projName != "" {
		title = fmt.Sprintf(" Conversation · %s", projName)
		if worktree != "" {
			title += fmt.Sprintf(" · ⎇ %s", worktree)
		}
	}
	right := lipgloss.NewStyle().
		Background(lipgloss.Color(ui.ColorHeaderFooter)).
		Foreground(lipgloss.Color(ui.ColorMuted)).
		Width(rightW + 1).
		Render(title)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func renderDivider(height int) string {
	bar := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorBorder)).Render("│")
	lines := make([]string, height)
	for i := range lines {
		lines[i] = bar
	}
	return strings.Join(lines, "\n")
}

func renderFatal(msg string, w, h int) string {
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	pad := (w - lipgloss.Width(msg)) / 2
	if pad < 0 {
		pad = 0
	}
	lines := make([]string, h)
	for i := range lines {
		if i == h/2 {
			lines[i] = strings.Repeat(" ", pad) + msg
		} else {
			lines[i] = strings.Repeat(" ", w)
		}
	}
	return strings.Join(lines, "\n")
}

func groupAndMerge(sessions []*session.Session, old []session.ProjectGroup) []session.ProjectGroup {
	collapseState := make(map[string]bool)
	for _, g := range old {
		collapseState[g.Name] = g.Collapsed
	}
	seen := make(map[string]int)
	var groups []session.ProjectGroup
	for _, s := range sessions {
		idx, ok := seen[s.ProjectName]
		if !ok {
			idx = len(groups)
			seen[s.ProjectName] = idx
			groups = append(groups, session.ProjectGroup{
				Name:      s.ProjectName,
				Collapsed: collapseState[s.ProjectName],
			})
		}
		groups[idx].Sessions = append(groups[idx].Sessions, s)
	}
	return groups
}

func flatCount(groups []session.ProjectGroup, collapsed map[string]bool) int {
	n := 0
	for _, g := range groups {
		n++
		if !collapsed[g.Name] {
			n += len(g.Sessions)
		}
	}
	return n
}
