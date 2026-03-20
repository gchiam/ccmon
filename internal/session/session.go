// internal/session/session.go
package session

import (
	"encoding/json"
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
// replace every '/', '_', and '.' with '-'.
func encodeCwd(cwd string) string {
	s := strings.ReplaceAll(cwd, "/", "-")
	s = strings.ReplaceAll(s, "_", "-")
	return strings.ReplaceAll(s, ".", "-")
}

// FindJSONLPath resolves the JSONL path for a session using the real projects dir.
func FindJSONLPath(cwd, sessionID string) string {
	return FindJSONLPathIn(projectsDir(), cwd, sessionID)
}

// FindJSONLPathIn is the testable variant that accepts an explicit projects directory.
// It encodes the cwd with the same algorithm Claude Code uses, then looks for that
// directory name. This avoids the fragility of reverse-decoding directory names.
//
// Lookup order:
//  1. Exact match: <baseDir>/<encoded>/<sessionID>.jsonl
//  2. sessions-index.json lookup in the project directory
//  3. Most recently modified .jsonl in the project directory
func FindJSONLPathIn(baseDir, cwd, sessionID string) string {
	encoded := encodeCwd(cwd)
	if encoded == "" {
		return ""
	}
	projDir := filepath.Join(baseDir, encoded)

	// 1. Exact match.
	candidate := filepath.Join(projDir, sessionID+".jsonl")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}

	// 2. Check sessions-index.json for the sessionID.
	if path := lookupSessionsIndex(projDir, sessionID); path != "" {
		return path
	}

	// 3. Fall back to the most recently modified .jsonl in the project directory.
	return mostRecentJSONL(projDir)
}

// sessionsIndex mirrors the relevant fields of ~/.claude/projects/<proj>/sessions-index.json.
type sessionsIndex struct {
	Entries []struct {
		SessionID string `json:"sessionId"`
		FullPath  string `json:"fullPath"`
	} `json:"entries"`
}

func lookupSessionsIndex(projDir, sessionID string) string {
	data, err := os.ReadFile(filepath.Join(projDir, "sessions-index.json"))
	if err != nil {
		return ""
	}
	var idx sessionsIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return ""
	}
	for _, e := range idx.Entries {
		if e.SessionID == sessionID && e.FullPath != "" {
			if _, err := os.Stat(e.FullPath); err == nil {
				return e.FullPath
			}
		}
	}
	return ""
}

func mostRecentJSONL(projDir string) string {
	entries, err := os.ReadDir(projDir)
	if err != nil {
		return ""
	}
	var best string
	var bestTime int64
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if t := info.ModTime().UnixNano(); t > bestTime {
			bestTime = t
			best = filepath.Join(projDir, e.Name())
		}
	}
	return best
}
