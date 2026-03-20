// internal/watcher/watcher_test.go
package watcher_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gchiam/ccmon/internal/session"
	"github.com/gchiam/ccmon/internal/watcher"
)

func TestLoadSessions_ReadsValidFiles(t *testing.T) {
	dir := t.TempDir()
	data := map[string]interface{}{
		"pid": 99999, "sessionId": "test-id", "cwd": "/tmp/myproj", "startedAt": 1000,
	}
	b, _ := json.Marshal(data)
	os.WriteFile(filepath.Join(dir, "99999.json"), b, 0644)

	sessions, err := watcher.LoadSessions(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1, got %d", len(sessions))
	}
	if sessions[0].SessionID != "test-id" {
		t.Errorf("got %q", sessions[0].SessionID)
	}
	if sessions[0].ProjectName != "myproj" {
		t.Errorf("got %q", sessions[0].ProjectName)
	}
}

func TestLoadSessions_SkipsMalformed(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "bad.json"), []byte("not json"), 0644)
	sessions, err := watcher.LoadSessions(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected 0, got %d", len(sessions))
	}
}

func TestLoadSessions_MissingDir(t *testing.T) {
	_, err := watcher.LoadSessions("/nonexistent/path/xyz")
	if err == nil {
		t.Error("expected error for missing dir")
	}
}

func TestGroupByProject(t *testing.T) {
	sessions := []*session.Session{
		{ProjectName: "proj-a", SessionID: "1"},
		{ProjectName: "proj-b", SessionID: "2"},
		{ProjectName: "proj-a", SessionID: "3"},
	}
	groups := watcher.GroupByProject(sessions)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	for _, g := range groups {
		if g.Name == "proj-a" && len(g.Sessions) != 2 {
			t.Errorf("proj-a: expected 2, got %d", len(g.Sessions))
		}
	}
}

// TestWatcher_StartsAndStops verifies that the watcher starts and can be stopped cleanly.
func TestWatcher_StartsAndStops(t *testing.T) {
	dir := t.TempDir()
	w := watcher.New(dir, nil)
	done := make(chan struct{})
	go func() {
		w.Run()
		close(done)
	}()
	time.Sleep(50 * time.Millisecond)
	w.Stop()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("watcher did not stop in time")
	}
}
