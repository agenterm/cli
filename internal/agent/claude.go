package agent

import "github.com/agenterm/cli/internal/gate"

// ClaudePreToolUseOutputter formats decisions for Claude Code PreToolUse hooks.
type ClaudePreToolUseOutputter struct{}

func (ClaudePreToolUseOutputter) Allow(reason string) interface{} {
	return gate.BuildHookOutput("allow", reason)
}

func (ClaudePreToolUseOutputter) Deny(reason string) interface{} {
	return gate.BuildHookOutput("deny", reason)
}

// ClaudePermissionOutputter formats decisions for Claude Code PermissionRequest hooks.
type ClaudePermissionOutputter struct{}

func (ClaudePermissionOutputter) Allow(reason string) interface{} {
	return gate.BuildPermissionRequestOutput("allow", reason)
}

func (ClaudePermissionOutputter) Deny(reason string) interface{} {
	return gate.BuildPermissionRequestOutput("deny", reason)
}
