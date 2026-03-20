package agent

// ClaudePreToolUseOutputter formats decisions for Claude Code PreToolUse hooks.
type ClaudePreToolUseOutputter struct{}

func (ClaudePreToolUseOutputter) Allow(reason string) interface{} {
	return BuildPreToolUseOutput("allow", reason)
}

func (ClaudePreToolUseOutputter) Deny(reason string) interface{} {
	return BuildPreToolUseOutput("deny", reason)
}

// ClaudePermissionOutputter formats decisions for Claude Code PermissionRequest hooks.
type ClaudePermissionOutputter struct{}

func (ClaudePermissionOutputter) Allow(reason string) interface{} {
	return BuildPermissionRequestOutput("allow", reason)
}

func (ClaudePermissionOutputter) Deny(reason string) interface{} {
	return BuildPermissionRequestOutput("deny", reason)
}

// PreToolUseOutput represents the JSON response for Claude Code PreToolUse hooks.
type PreToolUseOutput struct {
	HookSpecificOutput PreToolUseSpecificOutput `json:"hookSpecificOutput"`
}

// PreToolUseSpecificOutput contains the permission decision for a PreToolUse hook.
type PreToolUseSpecificOutput struct {
	HookEventName            string `json:"hookEventName"`
	PermissionDecision       string `json:"permissionDecision"`
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
}

// BuildPreToolUseOutput creates a PreToolUseOutput with the given decision and reason.
func BuildPreToolUseOutput(decision, reason string) *PreToolUseOutput {
	return &PreToolUseOutput{
		HookSpecificOutput: PreToolUseSpecificOutput{
			HookEventName:            "PreToolUse",
			PermissionDecision:       decision,
			PermissionDecisionReason: reason,
		},
	}
}

// PermissionRequestOutput represents the JSON response for Claude Code PermissionRequest hooks.
type PermissionRequestOutput struct {
	HookSpecificOutput PermissionRequestSpecificOutput `json:"hookSpecificOutput"`
}

// PermissionRequestSpecificOutput contains the decision for a PermissionRequest hook.
type PermissionRequestSpecificOutput struct {
	HookEventName string                    `json:"hookEventName"`
	Decision      PermissionRequestDecision `json:"decision"`
}

// PermissionRequestDecision holds the behavior and message for a PermissionRequest response.
type PermissionRequestDecision struct {
	Behavior string `json:"behavior"`
	Message  string `json:"message,omitempty"`
}

// BuildPermissionRequestOutput creates a PermissionRequestOutput with the given behavior and message.
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
