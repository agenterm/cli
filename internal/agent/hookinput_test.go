package agent

import (
	"strings"
	"testing"
)

func TestParseHookInput_Valid(t *testing.T) {
	input := `{"session_id":"s1","hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"rm -rf /"},"tool_use_id":"t1"}`
	h := ParseHookInput([]byte(input))
	if h == nil {
		t.Fatal("expected non-nil HookInput")
	}
	if h.ToolName != "Bash" {
		t.Fatalf("expected ToolName=Bash, got %q", h.ToolName)
	}
}

func TestParseHookInput_PermissionRequest(t *testing.T) {
	input := `{"session_id":"s2","hook_event_name":"PermissionRequest","tool_name":"Bash","tool_input":{"command":"rm -rf /tmp"},"tool_use_id":"t2"}`
	h := ParseHookInput([]byte(input))
	if h == nil {
		t.Fatal("expected non-nil HookInput for PermissionRequest")
	}
	if h.HookEventName != "PermissionRequest" {
		t.Fatalf("expected HookEventName=PermissionRequest, got %q", h.HookEventName)
	}
	if h.ToolName != "Bash" {
		t.Fatalf("expected ToolName=Bash, got %q", h.ToolName)
	}
}

func TestParseHookInput_NotSupported(t *testing.T) {
	input := `{"hook_event_name":"PostToolUse","tool_name":"Bash"}`
	h := ParseHookInput([]byte(input))
	if h != nil {
		t.Fatal("expected nil for unsupported event")
	}
}

func TestParseHookInput_InvalidJSON(t *testing.T) {
	h := ParseHookInput([]byte("not json"))
	if h != nil {
		t.Fatal("expected nil for invalid JSON")
	}
}

func TestExtractCheckInput_Bash(t *testing.T) {
	h := &HookInput{
		ToolName:  "Bash",
		ToolInput: map[string]interface{}{"command": "git push --force origin main"},
	}
	got := ExtractCheckInput(h)
	if got != "git push --force origin main" {
		t.Fatalf("expected command string, got %q", got)
	}
}

func TestExtractCheckInput_NonBash(t *testing.T) {
	h := &HookInput{
		ToolName:  "Write",
		ToolInput: map[string]interface{}{"file_path": "/tmp/x.sql", "content": "DROP TABLE users"},
	}
	got := ExtractCheckInput(h)
	if got == "" {
		t.Fatal("expected non-empty serialized input")
	}
	if !strings.Contains(got, "DROP TABLE") {
		t.Fatalf("expected serialized input to contain DROP TABLE, got %q", got)
	}
}

func TestExtractCheckInput_BashMissingCommand(t *testing.T) {
	h := &HookInput{
		ToolName:  "Bash",
		ToolInput: map[string]interface{}{},
	}
	got := ExtractCheckInput(h)
	// Falls through to JSON serialization
	if got == "" {
		t.Fatal("expected non-empty fallback")
	}
}
