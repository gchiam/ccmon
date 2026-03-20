// internal/ui/conversation.go
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	humanPrefix     = lipgloss.NewStyle().Foreground(SelectedSess).Bold(true).Render("▶")
	assistantPrefix = lipgloss.NewStyle().Foreground(ActiveSession).Bold(true).Render("◀")
	toolPrefix      = lipgloss.NewStyle().Foreground(UnreadTool).Bold(true).Render("⚙")
	resultPrefix    = lipgloss.NewStyle().Foreground(Muted).Render("→")

	codeBlockStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1e2030")).
			Foreground(lipgloss.Color("#c6d0f5")).
			Padding(0, 1)

	convPanelStyle = lipgloss.NewStyle().Background(PanelBg)
	tsStyle        = lipgloss.NewStyle().Foreground(Dim)
)

// RenderConversation renders the right panel.
func RenderConversation(entries []ConversationEntry, scroll, width, height int) string {
	if len(entries) == 0 {
		return renderEmpty(width, height)
	}

	var all []string
	for _, e := range entries {
		all = append(all, renderEntry(e, width)...)
		all = append(all, "")
	}

	maxScroll := len(all) - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	end := scroll + height
	if end > len(all) {
		end = len(all)
	}
	visible := all[scroll:end]
	for len(visible) < height {
		visible = append(visible, strings.Repeat(" ", width))
	}
	return convPanelStyle.Render(strings.Join(visible, "\n"))
}

func renderEntry(e ConversationEntry, width int) []string {
	switch e.Type {
	case "human":
		header := fmt.Sprintf("%s human · %s", humanPrefix, tsStyle.Render(e.Timestamp))
		return []string{header, WrapText(e.Text, width-2, "  ")}
	case "assistant":
		header := fmt.Sprintf("%s assistant · %s", assistantPrefix, tsStyle.Render(e.Timestamp))
		return []string{header, WrapText(e.Text, width-2, "  ")}
	case "tool_call":
		header := fmt.Sprintf("%s %s · %s", toolPrefix,
			lipgloss.NewStyle().Foreground(UnreadTool).Render(e.ToolName),
			tsStyle.Render(e.Timestamp))
		block := codeBlockStyle.Width(width - 4).Render(e.Text)
		return []string{header, block}
	case "tool_result":
		return []string{fmt.Sprintf("%s %s", resultPrefix, e.Result)}
	default:
		return nil
	}
}

func renderEmpty(width, height int) string {
	msg := "No conversation history yet"
	pad := (width - lipgloss.Width(msg)) / 2
	if pad < 0 {
		pad = 0
	}
	line := strings.Repeat(" ", pad) + mutedStyle.Render(msg)
	lines := make([]string, height)
	middle := height / 2
	for i := range lines {
		if i == middle {
			lines[i] = line
		} else {
			lines[i] = strings.Repeat(" ", width)
		}
	}
	return convPanelStyle.Render(strings.Join(lines, "\n"))
}

// WrapText wraps text at width characters, prefixing each line with indent.
// Exported for testing.
func WrapText(text string, width int, indent string) string {
	if width <= 0 {
		return indent + text
	}
	var out []string
	for _, para := range strings.Split(text, "\n") {
		runes := []rune(para)
		for len(runes) > width {
			out = append(out, indent+string(runes[:width]))
			runes = runes[width:]
		}
		out = append(out, indent+string(runes))
	}
	return strings.Join(out, "\n")
}
