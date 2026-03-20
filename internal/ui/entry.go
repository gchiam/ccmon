// internal/ui/entry.go
package ui

// ConversationEntry is a single parsed line from a session JSONL.
// Defined here (rather than in model) to avoid an import cycle between
// model -> ui -> model.
type ConversationEntry struct {
	Type      string // "human", "assistant", "tool_call", "tool_result"
	Timestamp string // formatted HH:MM:SS
	Text      string // message body or tool input JSON
	ToolName  string // only for tool_call
	Result    string // only for tool_result (truncated)
}
