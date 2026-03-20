package agent

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
