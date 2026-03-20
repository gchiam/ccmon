// cmd/ccmon/main.go
package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gchiam/ccmon/internal/model"
	"github.com/gchiam/ccmon/internal/reader"
	"github.com/gchiam/ccmon/internal/session"
	"github.com/gchiam/ccmon/internal/watcher"
)

func main() {
	sessDir := session.SessionsDir()
	if _, err := os.Stat(sessDir); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "Claude Code sessions directory not found. Is Claude Code installed? Press q to quit.")
		os.Exit(1)
	}

	m := model.New()
	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	w := watcher.New(sessDir, p)
	go w.Run()
	defer w.Stop()

	// readerCancel cancels the currently running reader goroutine.
	// A new reader is created fresh for each session selection (no reuse).
	var readerCancel context.CancelFunc

	m.OnSelectSession(func(s *session.Session) {
		// Cancel the previous reader goroutine if one is running.
		if readerCancel != nil {
			readerCancel()
			readerCancel = nil
		}
		if s == nil || s.JSONLPath == "" {
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		readerCancel = cancel
		// Always create a fresh Reader so there is no shared state between sessions.
		r := reader.New(s.JSONLPath, s.SessionID, p)
		go r.Run(ctx)
		w.SetSelected(s.SessionID)
	})

	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
