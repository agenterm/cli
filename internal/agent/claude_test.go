package agent

import (
	"encoding/json"
	"testing"
)

func TestBuildPreToolUseOutput(t *testing.T) {
	out := BuildPreToolUseOutput("deny", "too dangerous")
	if out.HookSpecificOutput.HookEventName != "PreToolUse" {
		t.Fatalf("expected hookEventName=PreToolUse, got %q", out.HookSpecificOutput.HookEventName)
	}
	if out.HookSpecificOutput.PermissionDecision != "deny" {
		t.Fatalf("expected deny, got %q", out.HookSpecificOutput.PermissionDecision)
	}
	if out.HookSpecificOutput.PermissionDecisionReason != "too dangerous" {
		t.Fatalf("expected reason, got %q", out.HookSpecificOutput.PermissionDecisionReason)
	}

	// Verify JSON marshaling
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var roundtrip PreToolUseOutput
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if roundtrip.HookSpecificOutput.PermissionDecision != "deny" {
		t.Fatal("roundtrip failed")
	}
}

func TestBuildPermissionRequestOutput(t *testing.T) {
	out := BuildPermissionRequestOutput("allow", "approved via AgenTerm")
	if out.HookSpecificOutput.HookEventName != "PermissionRequest" {
		t.Fatalf("expected hookEventName=PermissionRequest, got %q", out.HookSpecificOutput.HookEventName)
	}
	if out.HookSpecificOutput.Decision.Behavior != "allow" {
		t.Fatalf("expected behavior=allow, got %q", out.HookSpecificOutput.Decision.Behavior)
	}
	if out.HookSpecificOutput.Decision.Message != "approved via AgenTerm" {
		t.Fatalf("expected message, got %q", out.HookSpecificOutput.Decision.Message)
	}

	// Verify JSON structure matches Claude Code PermissionRequest format
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	hso, ok := raw["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("missing hookSpecificOutput")
	}
	if hso["hookEventName"] != "PermissionRequest" {
		t.Fatalf("wrong hookEventName in JSON: %v", hso["hookEventName"])
	}
	decision, ok := hso["decision"].(map[string]interface{})
	if !ok {
		t.Fatal("missing decision object")
	}
	if decision["behavior"] != "allow" {
		t.Fatalf("wrong behavior in JSON: %v", decision["behavior"])
	}
}

func TestBuildPermissionRequestOutput_Deny(t *testing.T) {
	out := BuildPermissionRequestOutput("deny", "denied via AgenTerm")
	if out.HookSpecificOutput.Decision.Behavior != "deny" {
		t.Fatalf("expected behavior=deny, got %q", out.HookSpecificOutput.Decision.Behavior)
	}
}
