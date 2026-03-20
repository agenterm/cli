package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/agenterm/cli/internal/agent"
	"github.com/agenterm/cli/internal/hook"
)

func runHook(args []string) int {
	if len(args) < 1 {
		printHookUsage()
		return 2
	}

	target := "all"
	if len(args) >= 2 {
		target = args[1]
	}

	switch args[0] {
	case "install":
		return runHookInstall(target)
	case "uninstall":
		return runHookUninstall(target)
	default:
		fmt.Fprintf(os.Stderr, "unknown hook command: %s\n", args[0])
		printHookUsage()
		return 2
	}
}

func printHookUsage() {
	fmt.Fprintln(os.Stderr, "Usage: agenterm hook <install|uninstall> [claude|gemini]")
	fmt.Fprintln(os.Stderr, "  If no target is specified, applies to all supported agents.")
}

func runHookInstall(target string) int {
	binaryPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot determine binary path: %v\n", err)
		return 1
	}

	return forEachTarget(target, func(t agent.HookTarget) (string, error) {
		if err := t.Install(binaryPath, ""); err != nil {
			if errors.Is(err, hook.ErrAlreadyInstalled) {
				return "hook already installed", nil
			}
			return "", fmt.Errorf("error: %v", err)
		}
		settingsPath, _ := t.SettingsPath()
		return fmt.Sprintf("Installed %s hook in %s", t.HookName, settingsPath), nil
	})
}

func runHookUninstall(target string) int {
	return forEachTarget(target, func(t agent.HookTarget) (string, error) {
		if err := t.Uninstall(""); err != nil {
			if errors.Is(err, hook.ErrNotInstalled) {
				return "hook not found", nil
			}
			return "", fmt.Errorf("error: %v", err)
		}
		settingsPath, _ := t.SettingsPath()
		return fmt.Sprintf("Uninstalled hook from %s", settingsPath), nil
	})
}

// forEachTarget iterates over agent targets matching the filter and runs fn.
// The callback returns a success message or an error for fatal failures.
func forEachTarget(target string, fn func(agent.HookTarget) (string, error)) int {
	matched := false
	exitCode := 0
	for _, t := range agent.Targets() {
		if target != "all" && target != t.Name {
			continue
		}
		matched = true
		msg, err := fn(t)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%s] %v\n", t.Name, err)
			exitCode = 1
			continue
		}
		if msg != "" {
			fmt.Fprintf(os.Stderr, "[%s] %s\n", t.Name, msg)
		}
	}
	if !matched {
		fmt.Fprintf(os.Stderr, "unknown target: %s (use claude, gemini, or omit for all)\n", target)
		return 2
	}
	return exitCode
}
