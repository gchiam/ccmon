// internal/model/messages.go
package model

import (
	"github.com/gchiam/ccmon/internal/session"
	"github.com/gchiam/ccmon/internal/ui"
)

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
	Entries   []ui.ConversationEntry
}

// WatchErrorMsg signals that fsnotify setup failed; the watcher falls back to polling.
type WatchErrorMsg struct {
	Err error
}
