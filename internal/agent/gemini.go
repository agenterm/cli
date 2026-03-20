package agent

import "github.com/agenterm/cli/internal/hook"

func init() {
	Register(
		HookTarget{Name: "gemini", HookName: "BeforeTool", Config: hook.GeminiHookConfig},
		GeminiOutputter{},
	)
}

// GeminiOutputter formats decisions for Gemini CLI BeforeTool hooks.
type GeminiOutputter struct{}

func (GeminiOutputter) Allow(_ string) interface{} {
	return &GeminiHookOutput{Decision: "allow"}
}

func (GeminiOutputter) Deny(reason string) interface{} {
	return &GeminiHookOutput{Decision: "deny", Reason: reason}
}

// GeminiHookOutput represents the JSON response for Gemini CLI BeforeTool hooks.
type GeminiHookOutput struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason,omitempty"`
}
