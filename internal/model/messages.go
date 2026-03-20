// internal/model/messages.go
package model

import "github.com/gchiam/ccmon/internal/session"

// SessionsUpdatedMsg is emitted by the watcher whenever the session list changes.
type SessionsUpdatedMsg struct {
	Sessions []*session.Session
}

// UnreadMsg is emitted by the watcher when a non-selected session's JSONL grows.
type UnreadMsg struct {
	SessionID string
}

// ConversationMsg is emitted by the reader when new entries are available.
type ConversationMsg struct {
	SessionID string
	Entries   []ConversationEntry
}

// ConversationEntry is a single parsed line from a session JSONL.
type ConversationEntry struct {
	Type      string // "human", "assistant", "tool_call", "tool_result"
	Timestamp string // formatted HH:MM:SS
	Text      string // message body or tool input JSON
	ToolName  string // only for tool_call
	Result    string // only for tool_result (truncated)
}

// WatchErrorMsg signals that fsnotify setup failed; the watcher falls back to polling.
type WatchErrorMsg struct {
	Err error
}
