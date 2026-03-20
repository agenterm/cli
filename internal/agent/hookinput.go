package agent

import "encoding/json"

// HookInput represents the JSON that AI agents pass to hooks via stdin.
type HookInput struct {
	SessionID     string                 `json:"session_id"`
	HookEventName string                 `json:"hook_event_name"`
	ToolName      string                 `json:"tool_name"`
	ToolInput     map[string]interface{} `json:"tool_input"`
	ToolUseID     string                 `json:"tool_use_id"`
}

// ParseHookInput attempts to parse raw bytes as a hook input from an agent.
// Returns nil if the input is not valid hook JSON or lacks a hook_event_name.
func ParseHookInput(data []byte) *HookInput {
	var h HookInput
	if err := json.Unmarshal(data, &h); err != nil {
		return nil
	}
	if h.HookEventName == "" {
		return nil
	}
	return &h
}

// ExtractCheckInput extracts the string to check against rules from a hook input.
// For Bash tools, extracts the command. For others, serializes tool_input.
func ExtractCheckInput(h *HookInput) string {
	if h.ToolName == "Bash" {
		if cmd, ok := h.ToolInput["command"].(string); ok {
			return cmd
		}
	}
	// For non-Bash tools, serialize all tool_input values to catch patterns
	// like DROP TABLE in Write/Edit content.
	data, err := json.Marshal(h.ToolInput)
	if err != nil {
		return ""
	}
	return string(data)
}
