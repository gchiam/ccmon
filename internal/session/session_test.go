// internal/session/session_test.go
package session_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gchiam/ccmon/internal/session"
)

func TestParseCwd_Plain(t *testing.T) {
	s := session.ParseCwd("/Users/alice/myproject")
	if s.ProjectName != "myproject" {
		t.Errorf("got %q", s.ProjectName)
	}
	if s.WorktreeBranch != "" {
		t.Errorf("expected empty worktree, got %q", s.WorktreeBranch)
	}
}

func TestParseCwd_Worktree(t *testing.T) {
	s := session.ParseCwd("/Users/alice/myproject--claude-worktrees-feat-login")
	if s.ProjectName != "myproject" {
		t.Errorf("got %q", s.ProjectName)
	}
	if s.WorktreeBranch != "feat-login" {
		t.Errorf("got %q", s.WorktreeBranch)
	}
}

func TestParseCwd_Empty(t *testing.T) {
	s := session.ParseCwd("")
	if s.ProjectName != "" {
		t.Errorf("got %q", s.ProjectName)
	}
}

func TestIsAlive_CurrentProcess(t *testing.T) {
	pid := os.Getpid()
	if !session.IsAlive(pid) {
		t.Errorf("expected current process (pid %d) to be alive", pid)
	}
}

func TestIsAlive_InvalidPID(t *testing.T) {
	if session.IsAlive(-1) {
		t.Error("expected pid -1 to be not alive")
	}
	if session.IsAlive(0) {
		t.Error("expected pid 0 to be not alive")
	}
}

func TestFindJSONLPath_Match(t *testing.T) {
	// Set up a fake ~/.claude/projects/ layout in a temp dir.
	// Encoding: strip leading '/', replace '/' with '-'.
	// cwd = "/Users/alice/myproject" -> encoded dir = "Users-alice-myproject"
	projectsDir := t.TempDir()
	encodedDir := "Users-alice-myproject"
	sessionID := "test-session-id"
	jsonlDir := filepath.Join(projectsDir, encodedDir)
	os.MkdirAll(jsonlDir, 0755)
	jsonlFile := filepath.Join(jsonlDir, sessionID+".jsonl")
	os.WriteFile(jsonlFile, []byte(""), 0644)

	got := session.FindJSONLPathIn(projectsDir, "/Users/alice/myproject", sessionID)
	if got != jsonlFile {
		t.Errorf("got %q, want %q", got, jsonlFile)
	}
}

func TestFindJSONLPath_DashInDirName(t *testing.T) {
	// cwd = "/Users/alice/my-project" -> encoded dir = "Users-alice-my-project"
	// A naive reverse-decode would mistake "my-project" as "my/project". This test
	// verifies the forward-encoding approach handles it correctly.
	projectsDir := t.TempDir()
	encodedDir := "Users-alice-my-project"
	sessionID := "sess-1"
	jsonlDir := filepath.Join(projectsDir, encodedDir)
	os.MkdirAll(jsonlDir, 0755)
	jsonlFile := filepath.Join(jsonlDir, sessionID+".jsonl")
	os.WriteFile(jsonlFile, []byte(""), 0644)

	got := session.FindJSONLPathIn(projectsDir, "/Users/alice/my-project", sessionID)
	if got != jsonlFile {
		t.Errorf("got %q, want %q", got, jsonlFile)
	}
}

func TestFindJSONLPath_NoMatch(t *testing.T) {
	projectsDir := t.TempDir()
	got := session.FindJSONLPathIn(projectsDir, "/Users/alice/nonexistent", "sess")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}
