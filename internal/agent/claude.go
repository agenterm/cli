package agent

import "github.com/agenterm/cli/internal/hook"

func init() {
	Register(
		HookTarget{Name: "claude", HookName: "PermissionRequest", Config: hook.ClaudeHookConfig},
		ClaudePermissionOutputter{},
	)

	// Register outputters for all decision hook events.
	for _, event := range DecisionHookEvents {
		switch event {
		case "PermissionRequest":
			// Already registered above.
		case "PreToolUse":
			registeredOutputters[event] = ClaudePreToolUseOutputter{}
		case "UserPromptSubmit":
			registeredOutputters[event] = ClaudeBlockOutputter{EventName: event}
		case "PostToolUse":
			registeredOutputters[event] = ClaudeBlockOutputter{EventName: event}
		case "Stop", "SubagentStop":
			registeredOutputters[event] = ClaudeBlockOutputter{EventName: event}
		case "ConfigChange":
			registeredOutputters[event] = ClaudeBlockOutputter{EventName: event}
		case "TaskCreated", "TaskCompleted", "TeammateIdle":
			registeredOutputters[event] = ExitCodeOutputter{}
		case "WorktreeCreate":
			registeredOutputters[event] = ExitCodeOutputter{}
		case "Elicitation":
			registeredOutputters[event] = ClaudeElicitationOutputter{EventName: "Elicitation"}
		case "ElicitationResult":
			registeredOutputters[event] = ClaudeElicitationOutputter{EventName: "ElicitationResult"}
		}
	}
}

// ---------------------------------------------------------------------------
// PreToolUse
// ---------------------------------------------------------------------------

// ClaudePreToolUseOutputter formats decisions for Claude Code PreToolUse hooks.
type ClaudePreToolUseOutputter struct{}

func (ClaudePreToolUseOutputter) Allow(reason string) interface{} {
	return BuildPreToolUseOutput("allow", reason)
}

func (ClaudePreToolUseOutputter) Deny(reason string) interface{} {
	return BuildPreToolUseOutput("deny", reason)
}

type PreToolUseOutput struct {
	HookSpecificOutput PreToolUseSpecificOutput `json:"hookSpecificOutput"`
}

type PreToolUseSpecificOutput struct {
	HookEventName            string `json:"hookEventName"`
	PermissionDecision       string `json:"permissionDecision"`
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
}

func BuildPreToolUseOutput(decision, reason string) *PreToolUseOutput {
	return &PreToolUseOutput{
		HookSpecificOutput: PreToolUseSpecificOutput{
			HookEventName:            "PreToolUse",
			PermissionDecision:       decision,
			PermissionDecisionReason: reason,
		},
	}
}

// ---------------------------------------------------------------------------
// PermissionRequest
// ---------------------------------------------------------------------------

// ClaudePermissionOutputter formats decisions for Claude Code PermissionRequest hooks.
type ClaudePermissionOutputter struct{}

func (ClaudePermissionOutputter) Allow(reason string) interface{} {
	return BuildPermissionRequestOutput("allow", reason)
}

func (ClaudePermissionOutputter) Deny(reason string) interface{} {
	return BuildPermissionRequestOutput("deny", reason)
}

type PermissionRequestOutput struct {
	HookSpecificOutput PermissionRequestSpecificOutput `json:"hookSpecificOutput"`
}

type PermissionRequestSpecificOutput struct {
	HookEventName string                    `json:"hookEventName"`
	Decision      PermissionRequestDecision `json:"decision"`
}

type PermissionRequestDecision struct {
	Behavior string `json:"behavior"`
	Message  string `json:"message,omitempty"`
}

func BuildPermissionRequestOutput(behavior, message string) *PermissionRequestOutput {
	return &PermissionRequestOutput{
		HookSpecificOutput: PermissionRequestSpecificOutput{
			HookEventName: "PermissionRequest",
			Decision: PermissionRequestDecision{
				Behavior: behavior,
				Message:  message,
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Block-style hooks: UserPromptSubmit, PostToolUse, Stop, SubagentStop, ConfigChange
// These use { decision: "block", reason: "..." } to deny, or empty output to allow.
// ---------------------------------------------------------------------------

// ClaudeBlockOutputter handles hooks that use decision:"block" to deny.
type ClaudeBlockOutputter struct {
	EventName string
}

func (o ClaudeBlockOutputter) Allow(_ string) interface{} {
	return nil // empty output = allow
}

func (o ClaudeBlockOutputter) Deny(reason string) interface{} {
	return map[string]interface{}{
		"decision": "block",
		"reason":   reason,
	}
}

// ---------------------------------------------------------------------------
// Exit-code hooks: TaskCreated, TaskCompleted, TeammateIdle, WorktreeCreate
// These use exit code 2 + stderr to block; exit 0 to allow. No JSON output.
// ---------------------------------------------------------------------------

// ExitCodeOutputter signals decisions via exit code rather than JSON.
// Allow returns nil (exit 0), Deny returns a special sentinel.
type ExitCodeOutputter struct{}

// ExitCodeDeny is a sentinel value indicating denial via exit code 2.
type ExitCodeDeny struct {
	Reason string
}

func (ExitCodeOutputter) Allow(_ string) interface{} {
	return nil
}

func (ExitCodeOutputter) Deny(reason string) interface{} {
	return &ExitCodeDeny{Reason: reason}
}

// ---------------------------------------------------------------------------
// Elicitation / ElicitationResult
// ---------------------------------------------------------------------------

// ClaudeElicitationOutputter handles Elicitation and ElicitationResult hooks.
type ClaudeElicitationOutputter struct {
	EventName string
}

func (o ClaudeElicitationOutputter) Allow(_ string) interface{} {
	return map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName": o.EventName,
			"action":        "accept",
		},
	}
}

func (o ClaudeElicitationOutputter) Deny(reason string) interface{} {
	return map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName": o.EventName,
			"action":        "decline",
		},
	}
}
