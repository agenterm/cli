package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/agenterm/cli/internal/agent"
	"github.com/agenterm/cli/internal/config"
	"github.com/agenterm/cli/internal/gate"
	"github.com/agenterm/cli/internal/relay"
)

func runGate(args []string) int {
	fs := flag.NewFlagSet("gate", flag.ContinueOnError)
	timeout := fs.Int("timeout", 60, "approval timeout in seconds")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
	}

	// APN enforces a minimum expires_in of 60s; clamp to avoid silent mismatch.
	if *timeout < 60 {
		*timeout = 60
	}

	// If positional args provided, use legacy mode (raw text).
	if remaining := fs.Args(); len(remaining) > 0 {
		return runGateLegacy(strings.Join(remaining, " "), *timeout)
	}

	// Otherwise, read stdin and auto-detect hook JSON vs raw text.
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
		return 2
	}

	if hookInput := agent.ParseHookInput(data); hookInput != nil {
		return runGateHook(hookInput, *timeout)
	}

	// Legacy mode: treat stdin as raw text.
	input := strings.TrimSpace(string(data))
	if input == "" {
		fmt.Fprintln(os.Stderr, "Usage: agenterm gate <tool_input>")
		return 2
	}
	return runGateLegacy(input, *timeout)
}

// runGateHook dispatches hook input to the appropriate handler based on event type.
func runGateHook(hookInput *agent.HookInput, timeout int) int {
	if hookInput.HookEventName == "PermissionRequest" {
		return runGatePermissionRequest(hookInput, timeout)
	}
	return runGateToolCheck(hookInput, timeout, agent.OutputterForEvent(hookInput.HookEventName))
}

// runGateToolCheck is the gate logic for rule-matched hooks (PreToolUse, BeforeTool).
func runGateToolCheck(hookInput *agent.HookInput, timeout int, out agent.Outputter) int {
	input := agent.ExtractCheckInput(hookInput)
	if input == "" {
		outputJSON(out.Allow("no input to check"))
		return 0
	}

	rules := gate.DefaultRules()
	matched, rule := gate.MatchesAny(input, rules)
	if !matched {
		outputJSON(out.Allow("no dangerous pattern matched"))
		return 0
	}

	fmt.Fprintf(os.Stderr, "matched rule: %s\n", rule.Description)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gate: config not found, blocking dangerous operation: %s\n", rule.Description)
		return 2
	}

	client := relay.NewClient(cfg)
	result, err := gate.RunGate(client, input, rules, time.Duration(timeout)*time.Second)
	denyDetail := fmt.Sprintf("denied via AgenTerm: %s", rule.Description)
	return executeGateCheck(result, err, out, input, denyDetail)
}

// runGatePermissionRequest handles PermissionRequest hooks.
// Claude Code has already determined this action needs permission — no rule matching needed.
func runGatePermissionRequest(hookInput *agent.HookInput, timeout int) int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gate: config not found, blocking permission request\n")
		return 2
	}

	title := hookInput.ToolName
	if title == "" {
		title = "Permission Request"
	}
	body := ""
	if len(hookInput.ToolInput) > 0 {
		data, err := json.Marshal(hookInput.ToolInput)
		if err == nil {
			body = string(data)
		}
	}

	client := relay.NewClient(cfg)
	out := agent.OutputterForEvent(hookInput.HookEventName)

	result, err := gate.RunPermissionGate(client, title, body, time.Duration(timeout)*time.Second)
	fallbackPrompt := fmt.Sprintf("Tool: %s\nInput: %s", title, body)
	return executeGateCheck(result, err, out, fallbackPrompt, "denied via AgenTerm")
}

// executeGateCheck handles the common result/error flow shared by all hook gate checks.
// fallbackPrompt is shown to the user when falling back to local terminal approval.
// denyDetail is the deny reason included in the output on denial.
func executeGateCheck(result *gate.GateResult, err error, out agent.Outputter, fallbackPrompt, denyDetail string) int {
	if errors.Is(err, relay.ErrGateDisabled) {
		fmt.Fprintf(os.Stderr, "gate: takeover not enabled, falling back to local prompt\n")
		if askLocallyInTerminal(fallbackPrompt) {
			outputJSON(out.Allow("approved locally via AgenTerm"))
		} else {
			outputJSON(out.Deny("denied locally via AgenTerm"))
		}
		return 0
	}
	if errors.Is(err, relay.ErrPushKeyExpired) {
		fmt.Fprintf(os.Stderr, "gate: push key expired or invalid — copy a fresh key from the AgenTerm app\n")
		fmt.Fprintf(os.Stderr, "gate: falling back to local prompt\n")
		if askLocallyInTerminal(fallbackPrompt) {
			outputJSON(out.Allow("approved locally via AgenTerm (push key expired)"))
		} else {
			outputJSON(out.Deny("denied locally via AgenTerm (push key expired)"))
		}
		return 0
	}
	if errors.Is(err, relay.ErrRateLimited) {
		fmt.Fprintf(os.Stderr, "gate: rate limited, denying for safety\n")
		outputJSON(out.Deny("rate limited by AgenTerm relay"))
		return 0
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "gate: error: %v\n", err)
		return 2
	}

	switch result.Decision {
	case "approved", "allow":
		outputJSON(out.Allow("approved via AgenTerm"))
	default:
		outputJSON(out.Deny(denyDetail))
	}
	return 0
}

// runGateLegacy handles the original gate behavior (raw text input).
func runGateLegacy(input string, timeout int) int {
	rules := gate.DefaultRules()
	matched, rule := gate.MatchesAny(input, rules)
	if !matched {
		fmt.Fprintln(os.Stderr, "no dangerous pattern matched, allowing")
		return 0
	}

	fmt.Fprintf(os.Stderr, "matched rule: %s\n", rule.Description)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		return 2
	}

	client := relay.NewClient(cfg)
	result, err := gate.RunGate(client, input, rules, time.Duration(timeout)*time.Second)
	if errors.Is(err, relay.ErrGateDisabled) {
		fmt.Fprintf(os.Stderr, "gate: takeover not enabled, falling back to local prompt\n")
		if askLocallyInTerminal(input) {
			return 0
		} else {
			return 1
		}
	}
	if errors.Is(err, relay.ErrRateLimited) {
		fmt.Fprintf(os.Stderr, "rate limited by relay server, denying for safety\n")
		return 1
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "error encoding output: %v\n", err)
		return 2
	}

	fmt.Fprintf(os.Stderr, "decision: %s\n", result.Decision)

	switch result.Decision {
	case "approved":
		return 0
	case "denied":
		return 1
	default:
		return 2
	}
}
