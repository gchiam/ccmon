// internal/reader/reader.go
package reader

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gchiam/ccmon/internal/model"
	"github.com/gchiam/ccmon/internal/ui"
)

// rawLine is the top-level structure of a JSONL entry.
type rawLine struct {
	Type      string     `json:"type"`
	Timestamp string     `json:"timestamp"`
	SessionID string     `json:"sessionId"`
	Message   rawMessage `json:"message"`
}

type rawMessage struct {
	Role    string       `json:"role"`
	Content []rawContent `json:"content"`
}

type rawContent struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   []rawContent    `json:"content"`
}

// ParseLine parses a single JSONL line and returns a ConversationEntry.
// Returns ok=false for lines that should be skipped (hook events, malformed JSON).
func ParseLine(line string) (ui.ConversationEntry, bool) {
	var raw rawLine
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return ui.ConversationEntry{}, false
	}
	ts := formatTimestamp(raw.Timestamp)
	switch raw.Type {
	case "human":
		return parseHumanMessage(raw, ts)
	case "assistant":
		return parseAssistantMessage(raw, ts)
	default:
		return ui.ConversationEntry{}, false
	}
}

func parseHumanMessage(raw rawLine, ts string) (ui.ConversationEntry, bool) {
	for _, c := range raw.Message.Content {
		switch c.Type {
		case "text":
			return ui.ConversationEntry{Type: "human", Timestamp: ts, Text: c.Text}, true
		case "tool_result":
			result := extractToolResultText(c)
			return ui.ConversationEntry{Type: "tool_result", Timestamp: ts, Result: truncate(result, 120)}, true
		}
	}
	return ui.ConversationEntry{}, false
}

func parseAssistantMessage(raw rawLine, ts string) (ui.ConversationEntry, bool) {
	for _, c := range raw.Message.Content {
		switch c.Type {
		case "text":
			return ui.ConversationEntry{Type: "assistant", Timestamp: ts, Text: c.Text}, true
		case "tool_use":
			text := extractToolInput(c)
			return ui.ConversationEntry{Type: "tool_call", Timestamp: ts, ToolName: c.Name, Text: text}, true
		}
	}
	return ui.ConversationEntry{}, false
}

func extractToolInput(c rawContent) string {
	if c.Name == "Bash" {
		var input struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal(c.Input, &input); err == nil && input.Command != "" {
			return input.Command
		}
	}
	return string(c.Input)
}

func extractToolResultText(c rawContent) string {
	for _, inner := range c.Content {
		if inner.Type == "text" {
			return inner.Text
		}
	}
	return string(c.Input)
}

func formatTimestamp(iso string) string {
	t, err := time.Parse(time.RFC3339Nano, iso)
	if err != nil {
		return iso
	}
	return t.Format("15:04:05")
}

func truncate(s string, n int) string {
	s = strings.SplitN(s, "\n", 2)[0] // first line only
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

// Reader polls a JSONL file for new lines and emits ConversationMsg to the program.
// Each Reader instance is created fresh for a new session; never reused across sessions.
type Reader struct {
	path      string
	sessionID string
	offset    int64
	program   *tea.Program
}

// New creates a Reader for the given JSONL path.
func New(path, sessionID string, program *tea.Program) *Reader {
	return &Reader{path: path, sessionID: sessionID, program: program}
}

// Run starts polling until ctx is cancelled. It reads any existing content first,
// then polls every 500ms for new bytes.
func (r *Reader) Run(ctx context.Context) {
	r.poll() // read existing content immediately

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.poll()
		}
	}
}

func (r *Reader) poll() {
	f, err := os.Open(r.path)
	if err != nil {
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return
	}
	if info.Size() <= r.offset {
		return
	}

	if _, err := f.Seek(r.offset, 0); err != nil {
		return
	}

	buf := make([]byte, info.Size()-r.offset)
	n, err := f.Read(buf)
	if n == 0 || err != nil {
		return
	}
	buf = buf[:n]

	// Only process complete lines. Find the last newline and advance offset
	// only up to that point, leaving any partial final line for the next poll.
	lastNL := strings.LastIndex(string(buf), "\n")
	if lastNL < 0 {
		// No complete line yet; wait for more data.
		return
	}
	r.offset += int64(lastNL + 1)
	complete := buf[:lastNL]

	var entries []ui.ConversationEntry
	for _, line := range strings.Split(string(complete), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if entry, ok := ParseLine(line); ok {
			entries = append(entries, entry)
		}
	}
	if len(entries) > 0 {
		r.program.Send(model.ConversationMsg{SessionID: r.sessionID, Entries: entries})
	}
}
