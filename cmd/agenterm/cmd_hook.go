package main

import (
	"errors"
	"flag"
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

	switch args[0] {
	case "install":
		return runHookInstallCmd(args[1:])
	case "uninstall":
		return runHookUninstallCmd(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown hook command: %s\n", args[0])
		printHookUsage()
		return 2
	}
}

func printHookUsage() {
	fmt.Fprintln(os.Stderr, "Usage: agenterm hook <install|uninstall> [claude|gemini] [--events=all|decision|observability]")
	fmt.Fprintln(os.Stderr, "  If no target is specified, applies to all supported agents.")
	fmt.Fprintln(os.Stderr, "  --events flag only applies to Claude Code hooks (default: decision).")
}

func runHookInstallCmd(args []string) int {
	fs := flag.NewFlagSet("hook install", flag.ContinueOnError)
	eventsFlag := fs.String("events", "decision", "which hooks to install: all, decision, observability")

	// Extract target (positional) before flags.
	target := "all"
	var flagArgs []string
	for _, a := range args {
		if len(a) > 0 && a[0] == '-' {
			flagArgs = append(flagArgs, a)
		} else if target == "all" {
			target = a
		}
	}
	if err := fs.Parse(flagArgs); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
	}

	binaryPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot determine binary path: %v\n", err)
		return 1
	}

	return forEachTarget(target, func(t agent.HookTarget) (string, error) {
		if t.Name == "claude" {
			return installClaudeHooks(binaryPath, t, *eventsFlag)
		}
		// Non-Claude agents: single-event install as before.
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

func installClaudeHooks(binaryPath string, t agent.HookTarget, eventsMode string) (string, error) {
	var events []string
	switch eventsMode {
	case "all":
		events = agent.AllHookEvents
	case "decision":
		events = agent.DecisionHookEvents
	case "observability":
		events = agent.ObservabilityHookEvents
	default:
		return "", fmt.Errorf("unknown --events value: %s (use all, decision, or observability)", eventsMode)
	}

	installed, err := hook.InstallMultipleHooks(binaryPath, "", events, t.Config)
	if err != nil {
		if errors.Is(err, hook.ErrAlreadyInstalled) {
			return "all hooks already installed", nil
		}
		return "", fmt.Errorf("error: %v", err)
	}
	settingsPath, _ := t.SettingsPath()
	return fmt.Sprintf("Installed %d hook(s) in %s", installed, settingsPath), nil
}

func runHookUninstallCmd(args []string) int {
	target := "all"
	if len(args) >= 1 {
		target = args[0]
	}

	return forEachTarget(target, func(t agent.HookTarget) (string, error) {
		if t.Name == "claude" {
			err := hook.UninstallAllHooks("", agent.AllHookEvents, t.Config)
			if err != nil {
				if errors.Is(err, hook.ErrNotInstalled) {
					return "hook not found", nil
				}
				return "", fmt.Errorf("error: %v", err)
			}
			settingsPath, _ := t.SettingsPath()
			return fmt.Sprintf("Uninstalled hooks from %s", settingsPath), nil
		}
		// Non-Claude agents: single-event uninstall.
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
