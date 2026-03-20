// internal/session/session.go
package session

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Session holds the parsed contents of a ~/.claude/sessions/<pid>.json file.
type Session struct {
	PID            int    `json:"pid"`
	SessionID      string `json:"sessionId"`
	Cwd            string `json:"cwd"`
	StartedAt      int64  `json:"startedAt"`
	ProjectName    string // derived from Cwd
	WorktreeBranch string // derived from Cwd, empty if plain session
	Alive          bool   // result of kill -0
	JSONLPath      string // resolved path to conversation JSONL
}

// ProjectGroup is a named group of sessions for the left panel.
type ProjectGroup struct {
	Name      string
	Sessions  []*Session
	Collapsed bool
}

// CwdInfo holds the derived fields from a cwd path.
type CwdInfo struct {
	ProjectName    string
	WorktreeBranch string
}

// ParseCwd extracts the project name and optional worktree branch from a cwd path.
// Worktree paths contain "--claude-worktrees-" as a delimiter in the final segment.
func ParseCwd(cwd string) CwdInfo {
	if cwd == "" {
		return CwdInfo{}
	}
	base := filepath.Base(cwd)
	const delimiter = "--claude-worktrees-"
	if idx := strings.Index(base, delimiter); idx != -1 {
		return CwdInfo{
			ProjectName:    base[:idx],
			WorktreeBranch: base[idx+len(delimiter):],
		}
	}
	return CwdInfo{ProjectName: base}
}

// IsAlive returns true if the process with the given PID is reachable via kill -0.
func IsAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

// SessionsDir returns the path to ~/.claude/sessions/.
func SessionsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "sessions")
}

// projectsDir returns the path to ~/.claude/projects/.
func projectsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "projects")
}

// encodeCwd converts an absolute cwd path to the directory name used by Claude Code:
// replace every '/' and '_' with '-'.
func encodeCwd(cwd string) string {
	s := strings.ReplaceAll(cwd, "/", "-")
	return strings.ReplaceAll(s, "_", "-")
}

// FindJSONLPath resolves the JSONL path for a session using the real projects dir.
func FindJSONLPath(cwd, sessionID string) string {
	return FindJSONLPathIn(projectsDir(), cwd, sessionID)
}

// FindJSONLPathIn is the testable variant that accepts an explicit projects directory.
// It encodes the cwd with the same algorithm Claude Code uses, then looks for that
// directory name. This avoids the fragility of reverse-decoding directory names.
func FindJSONLPathIn(baseDir, cwd, sessionID string) string {
	encoded := encodeCwd(cwd)
	if encoded == "" {
		return ""
	}
	candidate := filepath.Join(baseDir, encoded, sessionID+".jsonl")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}
