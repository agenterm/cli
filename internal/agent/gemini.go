package agent

import "github.com/agenterm/cli/internal/gate"

// GeminiOutputter formats decisions for Gemini CLI BeforeTool hooks.
type GeminiOutputter struct{}

func (GeminiOutputter) Allow(_ string) interface{} {
	return &gate.GeminiHookOutput{Decision: "allow"}
}

func (GeminiOutputter) Deny(reason string) interface{} {
	return &gate.GeminiHookOutput{Decision: "deny", Reason: reason}
}
