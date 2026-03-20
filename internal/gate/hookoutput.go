package gate

// HookOutput represents the JSON response for Claude Code PreToolUse hooks.
type HookOutput struct {
	HookSpecificOutput HookSpecificOutput `json:"hookSpecificOutput"`
}

// HookSpecificOutput contains the permission decision for a PreToolUse hook.
type HookSpecificOutput struct {
	HookEventName            string `json:"hookEventName"`
	PermissionDecision       string `json:"permissionDecision"`
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
}

// BuildHookOutput creates a HookOutput for PreToolUse with the given decision and reason.
func BuildHookOutput(decision, reason string) *HookOutput {
	return &HookOutput{
		HookSpecificOutput: HookSpecificOutput{
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

// GeminiHookOutput represents the JSON response for Gemini CLI BeforeTool hooks.
type GeminiHookOutput struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason,omitempty"`
}
