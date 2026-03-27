package agent

import "github.com/agenterm/cli/internal/hook"

// DecisionHookEvents are Claude Code hook events that require user approval.
var DecisionHookEvents = []string{
	"UserPromptSubmit", "PreToolUse", "PermissionRequest", "PostToolUse",
	"Stop", "SubagentStop", "TaskCreated", "TaskCompleted", "ConfigChange",
	"Elicitation", "ElicitationResult", "WorktreeCreate", "TeammateIdle",
}

// ObservabilityHookEvents are Claude Code hook events that are fire-and-forget.
var ObservabilityHookEvents = []string{
	"SessionStart", "InstructionsLoaded", "Notification", "SubagentStart",
	"PostToolUseFailure", "StopFailure", "SessionEnd", "PreCompact",
	"PostCompact", "WorktreeRemove", "CwdChanged", "FileChanged",
}

// AllHookEvents is the full list of supported Claude Code hook events.
var AllHookEvents = append(append([]string{}, DecisionHookEvents...), ObservabilityHookEvents...)

// IsDecisionEvent returns true if the event requires user approval.
func IsDecisionEvent(name string) bool {
	for _, e := range DecisionHookEvents {
		if e == name {
			return true
		}
	}
	return false
}

// IsObservabilityEvent returns true if the event is fire-and-forget.
func IsObservabilityEvent(name string) bool {
	for _, e := range ObservabilityHookEvents {
		if e == name {
			return true
		}
	}
	return false
}

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
