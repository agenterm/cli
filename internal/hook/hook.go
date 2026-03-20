package hook

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrAlreadyInstalled is returned when the hook is already present in settings.
	ErrAlreadyInstalled = errors.New("already installed")
	// ErrNotInstalled is returned when the hook is not found in settings.
	ErrNotInstalled = errors.New("not installed")
)

const hookMarker = "agenterm gate"

// HookConfig describes how to install/uninstall a hook for a specific agent.
type HookConfig struct {
	DefaultSettingsPath func() (string, error)
	EventName           string   // primary hook event (e.g., "PermissionRequest")
	LegacyEvents        []string // additional events to clean up on install/uninstall
	Matcher             string   // hook matcher pattern
	Timeout             int      // hook timeout value
	HookName            string   // optional name field in hook entry
}

// ClaudeHookConfig is the hook configuration for Claude Code.
var ClaudeHookConfig = HookConfig{
	DefaultSettingsPath: SettingsPath,
	EventName:           "PermissionRequest",
	LegacyEvents:        []string{"PreToolUse"},
	Matcher:             "",
	Timeout:             120,
}

// GeminiHookConfig is the hook configuration for Gemini CLI.
var GeminiHookConfig = HookConfig{
	DefaultSettingsPath: GeminiSettingsPath,
	EventName:           "BeforeTool",
	Matcher:             "*",
	Timeout:             120000,
	HookName:            "agenterm-gate",
}

// SettingsPath returns the default path to Claude Code's settings.json.
func SettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

// GeminiSettingsPath returns the default path to Gemini CLI's settings.json.
func GeminiSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".gemini", "settings.json"), nil
}

// InstallHook adds the agenterm gate hook to an agent's settings file.
// binaryPath is the absolute path to the agenterm binary.
// settingsPath can be empty to use the config's default path.
func InstallHook(binaryPath, settingsPath string, cfg HookConfig) error {
	if settingsPath == "" {
		var err error
		settingsPath, err = cfg.DefaultSettingsPath()
		if err != nil {
			return err
		}
	}

	settings, err := readSettings(settingsPath)
	if err != nil {
		return err
	}

	hooks := getHooksMap(settings)

	// Check if already installed.
	entries := getHookEntries(hooks, cfg.EventName)
	if containsMarker(entries) {
		return ErrAlreadyInstalled
	}

	// Clean up legacy events.
	for _, legacy := range cfg.LegacyEvents {
		legacyEntries := getHookEntries(hooks, legacy)
		if len(legacyEntries) > 0 {
			filtered, removed := removeMarkerEntries(legacyEntries)
			if removed > 0 {
				if len(filtered) == 0 {
					delete(hooks, legacy)
				} else {
					hooks[legacy] = filtered
				}
			}
		}
	}

	// Add hook entry.
	entry := buildEntry(binaryPath, cfg)
	entries = append(entries, entry)
	hooks[cfg.EventName] = entries

	return writeSettings(settingsPath, settings)
}

// UninstallHook removes all agenterm gate hooks for the given configuration.
// settingsPath can be empty to use the config's default path.
func UninstallHook(settingsPath string, cfg HookConfig) error {
	if settingsPath == "" {
		var err error
		settingsPath, err = cfg.DefaultSettingsPath()
		if err != nil {
			return err
		}
	}
	eventNames := append([]string{cfg.EventName}, cfg.LegacyEvents...)
	return uninstallFromSettings(settingsPath, eventNames...)
}

// buildEntry creates a hook entry from the given configuration.
func buildEntry(binaryPath string, cfg HookConfig) map[string]interface{} {
	hookEntry := map[string]interface{}{
		"type":    "command",
		"command": binaryPath + " gate",
		"timeout": cfg.Timeout,
	}
	if cfg.HookName != "" {
		hookEntry["name"] = cfg.HookName
	}
	return map[string]interface{}{
		"matcher": cfg.Matcher,
		"hooks":   []interface{}{hookEntry},
	}
}

// readSettings reads and parses settings.json at the given path.
// Returns an empty map if the file does not exist.
func readSettings(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return settings, nil
}

// writeSettings writes settings back to the given path with indentation.
func writeSettings(path string, settings map[string]interface{}) error {
	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

// getHooksMap returns the "hooks" object from settings, creating it if needed.
func getHooksMap(settings map[string]interface{}) map[string]interface{} {
	if h, ok := settings["hooks"].(map[string]interface{}); ok {
		return h
	}
	h := map[string]interface{}{}
	settings["hooks"] = h
	return h
}

// getHookEntries returns the array for a given hook event name.
func getHookEntries(hooks map[string]interface{}, eventName string) []interface{} {
	if arr, ok := hooks[eventName].([]interface{}); ok {
		return arr
	}
	return nil
}

// containsMarker checks if any hook entry in the array contains the agenterm gate command.
func containsMarker(entries []interface{}) bool {
	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		if strings.Contains(string(data), hookMarker) {
			return true
		}
	}
	return false
}

// removeMarkerEntries removes hook entries that contain the agenterm gate command.
// Returns the filtered array and the count of removed entries.
func removeMarkerEntries(entries []interface{}) ([]interface{}, int) {
	var result []interface{}
	removed := 0
	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			result = append(result, entry)
			continue
		}
		if strings.Contains(string(data), hookMarker) {
			removed++
		} else {
			result = append(result, entry)
		}
	}
	return result, removed
}

// uninstallFromSettings removes agenterm gate hooks for the given event names.
func uninstallFromSettings(settingsPath string, eventNames ...string) error {
	settings, err := readSettings(settingsPath)
	if err != nil {
		return err
	}

	hooks := getHooksMap(settings)
	totalRemoved := 0

	for _, eventName := range eventNames {
		entries := getHookEntries(hooks, eventName)
		if len(entries) == 0 {
			continue
		}
		filtered, removed := removeMarkerEntries(entries)
		totalRemoved += removed
		if len(filtered) == 0 {
			delete(hooks, eventName)
		} else {
			hooks[eventName] = filtered
		}
	}

	if totalRemoved == 0 {
		return ErrNotInstalled
	}

	// Clean up empty hooks object.
	if len(hooks) == 0 {
		delete(settings, "hooks")
	}

	return writeSettings(settingsPath, settings)
}
