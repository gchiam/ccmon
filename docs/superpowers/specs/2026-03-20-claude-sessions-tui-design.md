# Claude Sessions TUI — Design Spec

**Date:** 2026-03-20
**Status:** Approved

---

## Overview

A terminal UI application written in Go that monitors all active Claude Code sessions on the local machine and displays their conversation history in real time. The app is a single self-contained binary with no external dependencies beyond the Claude Code data directory.

---

## Goals

- Show all active Claude Code sessions grouped by project, with worktree awareness
- Display conversation history (human turns, assistant turns, tool calls + results) for a selected session
- Update the session list instantly when sessions start or stop (fsnotify)
- Indicate when an unselected session receives new activity (in-memory, live only)
- Verify session liveness via PID check (not just file presence)

## Non-Goals

- Sending messages or interacting with sessions (future milestone)
- Persistent unread state across TUI restarts
- Remote/SSH sessions
- Non-Claude Code session sources

---

## Architecture

Single Go binary. Three concurrent layers communicate via Bubble Tea messages (channels):

```
┌─────────────────────────────────────────────────────┐
│                    main.go                          │
│  starts watcher + reader, hands model to Bubble Tea │
└──────────────┬──────────────────────────┬───────────┘
               │                          │
    ┌──────────▼──────────┐   ┌───────────▼──────────┐
    │   session watcher   │   │   conversation reader │
    │  (fsnotify loop)    │   │  (tail JSONL on select)│
    │                     │   │                       │
    │ emits SessionAdded  │   │ emits MessageAppended │
    │ SessionRemoved msgs │   │ msgs to TUI model     │
    └─────────────────────┘   └───────────────────────┘
               │                          │
    ┌──────────▼──────────────────────────▼───────────┐
    │              Bubble Tea model                   │
    │  Update() handles all msgs, View() renders TUI  │
    └─────────────────────────────────────────────────┘
```

### Packages

| Package | Responsibility |
|---|---|
| `cmd/claude-sessions/` | Entrypoint, wires components together |
| `internal/watcher` | fsnotify loop over `~/.claude/sessions/` and all session JSONL files, PID liveness, emits Bubble Tea msgs |
| `internal/reader` | Tails selected session JSONL by polling for new bytes at ~500ms intervals, parses entries into typed structs, emits msgs |
| `internal/model` | Bubble Tea model: state, `Update()`, `View()` |
| `internal/ui` | Rendering helpers for session list and conversation panel |

---

## Data Sources

### Session files

Location: `~/.claude/sessions/<pid>.json` — the filename is the PID (e.g. `58926.json`).

Each file contains:
```json
{ "pid": 58926, "sessionId": "uuid", "cwd": "/Users/...", "startedAt": 1234567890 }
```

Read the `pid` field from inside the file (not parsed from the filename) for safety in case the two diverge.

A session is **active** if its file exists AND `kill -0 <pid>` succeeds. Otherwise it is **stale**.

### Conversation history

Location: `~/.claude/projects/<encoded-cwd>/<sessionId>.jsonl`

The encoded `cwd` is derived by stripping the leading `/` from the absolute path and replacing all remaining `/` with `-`. For example, `/Users/alice/myproject` becomes `Users-alice-myproject`. Do not reconstruct this encoding from scratch — instead, list the actual directory names under `~/.claude/projects/` and match against the session's `cwd` field to find the correct directory (avoids encoding edge cases).

Each line is a JSON event. Relevant types:

- `type: "human"` — user message
- `type: "assistant"` — Claude response
- Tool use and tool result are embedded within assistant/human message content blocks

Only human messages, assistant messages, and tool call/result pairs are surfaced in the UI. Hook progress events are ignored.

---

## Layout

Vertical split: narrow collapsible tree on the left, conversation detail on the right.

```
┌─────────────────┬──────────────────────────────────────┐
│ SESSIONS (4)    │ CONVERSATION · project · ⎇ worktree  │
│                 │                                      │
│ PROJECT-NAME    │  ▶ human · 14:23:01                  │
│   ● main        │    message text                      │
│   ⎇ worktree ✦  │                                      │
│                 │  ⚙ Bash · 14:23:08                   │
│ OTHER-PROJECT   │  ┌─────────────────────────────┐     │
│ ▶ collapsed     │  │ ls ~/.claude/sessions/      │     │
│                 │  └─────────────────────────────┘     │
│                 │  → 40185.json, 41024.json...         │
│                 │                                      │
│                 │  ◀ assistant · 14:23:12              │
│                 │    response text                     │
├─────────────────┴──────────────────────────────────────┤
│ ↑↓ nav · Space collapse · Tab switch · q quit          │
└────────────────────────────────────────────────────────┘
```

### Session list (left panel)

- Projects as collapsible group headers (Space to toggle)
- Each session row shows: status dot, name or worktree branch (⎇ prefix), pid, start time
- Session name derived from `cwd`: last path segment for plain sessions; worktree branch name for worktree sessions
- **Unread indicator**: yellow dot prefix + highlighted row background when new JSONL lines arrive for an unselected session. Detected via fsnotify watching all session JSONL files (handled in `internal/watcher`). Cleared on selection. In-memory only — does not persist across TUI restarts.
- Stale sessions: dimmed row, red `⚠`, "process dead" label

### Conversation panel (right panel)

- Scrollable list of conversation entries
- Human turns: blue `▶` prefix, timestamp, message text
- Assistant turns: green `◀` prefix, timestamp, message text
- Tool calls: amber `⚙` prefix, tool name, timestamp; tool input rendered in a dark code block (for `Bash` this is the command string; for all other tools render the full input JSON); result on the next line as `→ ...` (truncated to one line if long)
- Empty state: centered "No conversation history yet" placeholder
- Tails live — new entries append as the session progresses (~500ms poll)

### Color palette

Catppuccin Frappé throughout:

| Element | Color |
|---|---|
| Background | `#303446` |
| Panel background | `#292c3c` |
| Header/footer | `#232634` |
| Border | `#414559` |
| Selected row | `#414559` |
| Project headers | `#ca9ee6` (mauve) |
| Active session | `#a6d189` (green) |
| Selected session | `#8caaee` (blue) |
| Unread/tool call | `#e5c890` (yellow) |
| Stale/error | `#e78284` (red) |
| Muted text | `#949cbb` |
| Dim text | `#626880` |

---

## Session Detection

Initial focus on startup is the left panel (session list).

1. On startup: read all `~/.claude/sessions/*.json` files
2. fsnotify watches the directory for `CREATE`/`REMOVE` events
3. Each session: parse file, check `kill -0 <pid>` for liveness
4. Parse project name and worktree from `cwd`:
   - Plain path: project = last segment, no worktree
   - Worktree path (contains `--claude-worktrees-`): split on that delimiter, extract project and worktree slug
5. Group sessions by project name into `[]ProjectGroup`

**Future improvement:** Show last-activity time derived from JSONL file modification time (see Future Improvements).

---

## Error Handling

| Scenario | Behaviour |
|---|---|
| Process dead but file present | Stale row: dimmed, red `⚠ process dead` |
| JSONL file missing | Right panel: "No conversation history yet" placeholder |
| JSONL line fails to parse | Skip line silently, continue reading |
| `~/.claude/sessions/` missing | Full-screen fatal error: "Claude Code sessions directory not found — is Claude Code installed? Press q to quit." |
| fsnotify watch fails | Fall back to 2s polling; amber status bar warning: "⚠ File watching unavailable — polling every 2s" |
| Terminal too small (<80×24) | Full-screen warning: "⚠ Terminal too small — resize to at least 80×24 to continue" |

---

## Keyboard Bindings

| Key | Action |
|---|---|
| `↑` / `↓` | Navigate session list (when left panel focused) or scroll conversation (when right panel focused) |
| `Space` | Collapse/expand project group (left panel only) |
| `Enter` | Select session and move focus to right panel |
| `Tab` | Switch focus between left and right panels |
| `q` / `Ctrl+C` | Quit |

---

## Testing

- Unit tests for `cwd` path parsing (project name + worktree extraction)
- Unit tests for JSONL parsing — each message type with real sample lines
- Unit tests for PID liveness check (mock `kill -0`)
- No TUI rendering tests (Bubble Tea `View()` output is not meaningfully assertable)

---

## Dependencies

| Library | Purpose |
|---|---|
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/lipgloss` | Styling and layout |
| `github.com/fsnotify/fsnotify` | File system watching |

---

## Future Improvements

- Replace JSONL polling with fsnotify watch on the selected session file
- Show last-activity time from JSONL file modification time
- Filter/search sessions by project name
- Interact with sessions (send messages via Claude Code IPC — pending CC support)
