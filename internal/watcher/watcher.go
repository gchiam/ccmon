// internal/watcher/watcher.go
package watcher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
	"github.com/gchiam/ccmon/internal/model"
	"github.com/gchiam/ccmon/internal/session"
)

// Watcher monitors the sessions directory and all known JSONL files.
type Watcher struct {
	dir          string
	program      *tea.Program
	stop         chan struct{}
	stopOnce     sync.Once
	fsw          *fsnotify.Watcher
	jsonlWatched map[string]string // JSONL path -> sessionID

	mu         sync.Mutex
	selectedID string // suppress unread for this session; guarded by mu
}

// New creates a Watcher for the given sessions directory. program may be nil for testing.
func New(dir string, program *tea.Program) *Watcher {
	return &Watcher{
		dir:          dir,
		program:      program,
		stop:         make(chan struct{}),
		jsonlWatched: make(map[string]string),
	}
}

// SetSelected updates which session is currently selected (suppresses unread for it).
func (w *Watcher) SetSelected(sessionID string) {
	w.mu.Lock()
	w.selectedID = sessionID
	w.mu.Unlock()
}

// Run blocks until Stop() is called. It tries fsnotify first; on failure falls back
// to 2s polling and sends WatchErrorMsg to the program so the UI can warn the user.
func (w *Watcher) Run() {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		w.sendWatchError(err)
		w.runPolling()
		return
	}
	w.fsw = fsw
	defer fsw.Close()

	if err := fsw.Add(w.dir); err != nil {
		w.sendWatchError(err)
		w.runPolling()
		return
	}

	sessions, _ := LoadSessions(w.dir)
	w.sendUpdate(sessions)
	w.syncJSONLWatches(sessions)

	pollTicker := time.NewTicker(2 * time.Second)
	defer pollTicker.Stop()

	for {
		select {
		case <-w.stop:
			return
		case event, ok := <-fsw.Events:
			if !ok {
				return
			}
			if isSessionFile(event.Name) {
				sessions, _ := LoadSessions(w.dir)
				w.sendUpdate(sessions)
				w.syncJSONLWatches(sessions)
			} else if id, watched := w.jsonlWatched[event.Name]; watched {
				w.mu.Lock()
				sel := w.selectedID
				w.mu.Unlock()
				if id != sel {
					w.send(model.UnreadMsg{SessionID: id})
				}
			}
		case <-fsw.Errors:
			// ignore individual watch errors
		case <-pollTicker.C:
			sessions, _ := LoadSessions(w.dir)
			w.sendUpdate(sessions)
		}
	}
}

func (w *Watcher) runPolling() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			sessions, _ := LoadSessions(w.dir)
			w.sendUpdate(sessions)
		}
	}
}

func (w *Watcher) send(msg tea.Msg) {
	if w.program != nil {
		w.program.Send(msg)
	}
}

func (w *Watcher) sendUpdate(sessions []*session.Session) {
	w.send(model.SessionsUpdatedMsg{Sessions: sessions})
}

func (w *Watcher) sendWatchError(err error) {
	w.send(model.WatchErrorMsg{Err: err})
}

func (w *Watcher) syncJSONLWatches(sessions []*session.Session) {
	if w.fsw == nil {
		return
	}
	current := make(map[string]string)
	for _, s := range sessions {
		if s.JSONLPath != "" {
			current[s.JSONLPath] = s.SessionID
		}
	}
	for path, id := range current {
		if _, watching := w.jsonlWatched[path]; !watching {
			_ = w.fsw.Add(path)
			w.jsonlWatched[path] = id
		}
	}
	for path := range w.jsonlWatched {
		if _, ok := current[path]; !ok {
			_ = w.fsw.Remove(path)
			delete(w.jsonlWatched, path)
		}
	}
}

// Stop signals the watcher to exit. Safe to call multiple times.
func (w *Watcher) Stop() {
	w.stopOnce.Do(func() { close(w.stop) })
}

// LoadSessions reads all *.json files in dir, parses them, and checks PID liveness.
func LoadSessions(dir string) ([]*session.Session, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var sessions []*session.Session
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		s, err := loadSessionFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func loadSessionFile(path string) (*session.Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s session.Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	info := session.ParseCwd(s.Cwd)
	s.ProjectName = info.ProjectName
	s.WorktreeBranch = info.WorktreeBranch
	s.Alive = session.IsAlive(s.PID)
	s.JSONLPath = session.FindJSONLPath(s.Cwd, s.SessionID)
	return &s, nil
}

// GroupByProject groups sessions by ProjectName, preserving insertion order.
func GroupByProject(sessions []*session.Session) []session.ProjectGroup {
	seen := make(map[string]int)
	var groups []session.ProjectGroup
	for _, s := range sessions {
		idx, ok := seen[s.ProjectName]
		if !ok {
			idx = len(groups)
			seen[s.ProjectName] = idx
			groups = append(groups, session.ProjectGroup{Name: s.ProjectName})
		}
		groups[idx].Sessions = append(groups[idx].Sessions, s)
	}
	return groups
}

func isSessionFile(path string) bool {
	return strings.HasSuffix(path, ".json")
}
