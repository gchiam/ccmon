// internal/ui/session_list_test.go
package ui_test

import (
	"testing"

	"github.com/gchiam/ccmon/internal/session"
	"github.com/gchiam/ccmon/internal/ui"
)

func TestCountSessions(t *testing.T) {
	groups := []session.ProjectGroup{
		{Name: "a", Sessions: []*session.Session{{}, {}}},
		{Name: "b", Sessions: []*session.Session{{}}},
	}
	got := ui.CountSessions(groups)
	if got != 3 {
		t.Errorf("got %d, want 3", got)
	}
}

func TestWrapText_ShortLine(t *testing.T) {
	got := ui.WrapText("hello", 80, "  ")
	if got != "  hello" {
		t.Errorf("got %q", got)
	}
}

func TestWrapText_LongLine(t *testing.T) {
	got := ui.WrapText("abcdef", 3, "")
	// Should wrap into 2 lines: "abc" and "def"
	if got != "abc\ndef" {
		t.Errorf("got %q", got)
	}
}
