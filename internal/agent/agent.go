package agent

import "github.com/agenterm/cli/internal/hook"

// Outputter formats gate decisions for a specific agent's hook protocol.
type Outputter interface {
	Allow(reason string) interface{}
	Deny(reason string) interface{}
}

// HookTarget defines install/uninstall operations for a supported agent.
type HookTarget struct {
	Name     string
	HookName string
	Config   hook.HookConfig
}

// Install adds the agenterm gate hook to the agent's settings.
func (t HookTarget) Install(binaryPath, settingsPath string) error {
	return hook.InstallHook(binaryPath, settingsPath, t.Config)
}

// Uninstall removes the agenterm gate hook from the agent's settings.
func (t HookTarget) Uninstall(settingsPath string) error {
	return hook.UninstallHook(settingsPath, t.Config)
}

// SettingsPath returns the default path to the agent's settings file.
func (t HookTarget) SettingsPath() (string, error) {
	return t.Config.DefaultSettingsPath()
}

var (
	registeredTargets    []HookTarget
	registeredOutputters = map[string]Outputter{}
)

// Register adds an agent target and its event outputter to the registry.
// Each agent should call this from init() to self-register.
func Register(t HookTarget, out Outputter) {
	registeredTargets = append(registeredTargets, t)
	registeredOutputters[t.HookName] = out
}

// Targets returns all registered agent hook targets.
func Targets() []HookTarget {
	return registeredTargets
}

// OutputterForEvent returns the appropriate Outputter for a given hook event name.
// Defaults to Claude PreToolUse format for unknown events.
func OutputterForEvent(eventName string) Outputter {
	if o, ok := registeredOutputters[eventName]; ok {
		return o
	}
	return ClaudePreToolUseOutputter{}
}
