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
