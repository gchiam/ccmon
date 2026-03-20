// internal/reader/reader_test.go
package reader_test

import (
	"testing"

	"github.com/gchiam/ccmon/internal/reader"
)

func TestParseHumanLine(t *testing.T) {
	line := `{"type":"human","message":{"role":"user","content":[{"type":"text","text":"hello world"}]},"timestamp":"2024-01-01T14:23:01.000Z","sessionId":"abc"}`
	entry, ok := reader.ParseLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if entry.Type != "human" {
		t.Errorf("type: got %q", entry.Type)
	}
	if entry.Text != "hello world" {
		t.Errorf("text: got %q", entry.Text)
	}
	if entry.Timestamp != "14:23:01" {
		t.Errorf("ts: got %q", entry.Timestamp)
	}
}

func TestParseAssistantLine(t *testing.T) {
	line := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi there"}]},"timestamp":"2024-01-01T14:23:05.000Z","sessionId":"abc"}`
	entry, ok := reader.ParseLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if entry.Type != "assistant" {
		t.Errorf("type: got %q", entry.Type)
	}
	if entry.Text != "hi there" {
		t.Errorf("text: got %q", entry.Text)
	}
}

func TestParseToolUseLine_Bash(t *testing.T) {
	line := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{"command":"ls ~/.claude/sessions/"}}]},"timestamp":"2024-01-01T14:23:08.000Z","sessionId":"abc"}`
	entry, ok := reader.ParseLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if entry.Type != "tool_call" {
		t.Errorf("type: got %q", entry.Type)
	}
	if entry.ToolName != "Bash" {
		t.Errorf("tool: got %q", entry.ToolName)
	}
	if entry.Text != "ls ~/.claude/sessions/" {
		t.Errorf("cmd: got %q", entry.Text)
	}
}

func TestParseToolUseLine_NonBash(t *testing.T) {
	line := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/foo/bar.go"}}]},"timestamp":"2024-01-01T14:23:09.000Z","sessionId":"abc"}`
	entry, ok := reader.ParseLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if entry.Type != "tool_call" {
		t.Errorf("type: got %q", entry.Type)
	}
	if entry.ToolName != "Read" {
		t.Errorf("tool: got %q", entry.ToolName)
	}
	// Non-Bash tools render full input JSON.
	if entry.Text == "" {
		t.Error("expected non-empty input JSON")
	}
}

func TestParseToolResultLine(t *testing.T) {
	line := `{"type":"human","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"x","content":[{"type":"text","text":"file1.txt\nfile2.txt"}]}]},"timestamp":"2024-01-01T14:23:10.000Z","sessionId":"abc"}`
	entry, ok := reader.ParseLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if entry.Type != "tool_result" {
		t.Errorf("type: got %q", entry.Type)
	}
	if entry.Result == "" {
		t.Error("expected non-empty result")
	}
}

func TestParseIgnoresHookEvents(t *testing.T) {
	line := `{"type":"hook","event":"PreToolUse","sessionId":"abc"}`
	_, ok := reader.ParseLine(line)
	if ok {
		t.Error("hook events should be ignored (ok=false)")
	}
}

func TestParseMalformedLine(t *testing.T) {
	_, ok := reader.ParseLine("not json {{{{")
	if ok {
		t.Error("malformed JSON should return ok=false")
	}
}
