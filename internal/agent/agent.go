package agent

import "github.com/agenterm/cli/internal/hook"

// Outputter formats gate decisions for a specific agent's hook protocol.
type Outputter interface {
	Allow(reason string) interface{}
	Deny(reason string) interface{}
}

// HookTarget defines install/uninstall operations for a supported agent.
type HookTarget struct {
	Name         string
	HookName     string
	Install      func(binaryPath, settingsPath string) error
	Uninstall    func(settingsPath string) error
	SettingsPath func() (string, error)
}

// Targets returns all supported agent hook targets.
func Targets() []HookTarget {
	return []HookTarget{
		{Name: "claude", HookName: "PermissionRequest", Install: hook.Install, Uninstall: hook.Uninstall, SettingsPath: hook.SettingsPath},
		{Name: "gemini", HookName: "BeforeTool", Install: hook.InstallGemini, Uninstall: hook.UninstallGemini, SettingsPath: hook.GeminiSettingsPath},
	}
}

// OutputterForEvent returns the appropriate Outputter for a given hook event name.
// Defaults to Claude PreToolUse format for unknown events.
func OutputterForEvent(eventName string) Outputter {
	if o, ok := outputters[eventName]; ok {
		return o
	}
	return ClaudePreToolUseOutputter{}
}

var outputters = map[string]Outputter{
	"BeforeTool": GeminiOutputter{},
}
