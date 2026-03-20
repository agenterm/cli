package main

import (
	"bufio"
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
	"github.com/agenterm/cli/internal/hook"
	"github.com/agenterm/cli/internal/relay"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Println(version)
		os.Exit(0)
	case "init":
		os.Exit(runInit(os.Args[2:]))
	case "propose":
		os.Exit(runPropose(os.Args[2:]))
	case "proposal":
		os.Exit(runProposal(os.Args[2:]))
	case "gate":
		os.Exit(runGate(os.Args[2:]))
	case "hook":
		os.Exit(runHook(os.Args[2:]))
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: agenterm <command> [args]")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  init      [--push-key KEY] [--relay-url URL]")
	fmt.Fprintln(os.Stderr, "  version   Print version and exit")
	fmt.Fprintln(os.Stderr, "  propose   --title \"...\" --body \"...\" [--wait] [--timeout 60]")
	fmt.Fprintln(os.Stderr, "  proposal  status <proposal_id>")
	fmt.Fprintln(os.Stderr, "  hook      install [claude|gemini]")
	fmt.Fprintln(os.Stderr, "  hook      uninstall [claude|gemini]")
	fmt.Fprintln(os.Stderr, "  gate      (internal) used by hooks, not for direct use")
}

func runInit(args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	pushKey := fs.String("push-key", "", "push key from AgenTerm app")
	relayURL := fs.String("relay-url", "", "relay server URL")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
	}

	cfg := &config.Config{
		RelayURL: *relayURL,
		PushKey:  *pushKey,
	}

	// Interactive mode when --push-key is not provided
	if cfg.PushKey == "" {
		reader := bufio.NewReader(os.Stdin)

		fmt.Println("AgenTerm Agent Setup")
		fmt.Println()

		fmt.Printf("Relay URL (default: %s): ", config.DefaultRelayURL)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line != "" {
			cfg.RelayURL = line
		}

		fmt.Print("Push Key (from AgenTerm app > Account Details): ")
		line, _ = reader.ReadString('\n')
		cfg.PushKey = strings.TrimSpace(line)

		if cfg.PushKey == "" {
			fmt.Fprintln(os.Stderr, "error: push key is required")
			return 1
		}
	}

	if cfg.RelayURL == "" {
		cfg.RelayURL = config.DefaultRelayURL
	}

	fmt.Fprint(os.Stderr, "Validating push key... ")
	if err := relay.ValidatePushKey(cfg.RelayURL, cfg.PushKey); err != nil {
		fmt.Fprintf(os.Stderr, "\nerror: %v\n", err)
		return 1
	}
	fmt.Fprintln(os.Stderr, "ok")

	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "error saving config: %v\n", err)
		return 1
	}

	p, _ := config.ConfigPath()
	fmt.Fprintf(os.Stderr, "Configuration saved to %s\n", p)
	return 0
}

func runPropose(args []string) int {
	fs := flag.NewFlagSet("propose", flag.ContinueOnError)
	title := fs.String("title", "", "proposal title")
	body := fs.String("body", "", "proposal body")
	wait := fs.Bool("wait", true, "wait for proposal result")
	timeout := fs.Int("timeout", 60, "wait timeout in seconds")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
	}

	// Allow title as positional arg for convenience.
	if *title == "" {
		if remaining := fs.Args(); len(remaining) > 0 {
			*title = remaining[0]
		}
	}

	if *title == "" || *body == "" {
		fmt.Fprintln(os.Stderr, "Usage: agenterm propose --title \"...\" --body \"...\" [--wait] [--timeout 60]")
		fmt.Fprintln(os.Stderr, "       agenterm propose <title> --body \"...\"  (title as positional arg, flags must come first)")
		return 2
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		return 2
	}

	client := relay.NewClient(cfg)

	proposal, err := client.CreateProposal("approval", *title, *body, relay.WithExpiresIn(*timeout))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating proposal: %v\n", err)
		return 2
	}

	if !*wait {
		return outputProposal(proposal)
	}

	proposal, err = client.WaitForProposal(proposal.ID, time.Duration(*timeout)*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error waiting for proposal: %v\n", err)
		return 2
	}

	return outputProposal(proposal)
}

func runProposal(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: agenterm proposal status <proposal_id>")
		return 2
	}

	switch args[0] {
	case "status":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: agenterm proposal status <proposal_id>")
			return 2
		}
		return runProposalStatus(args[1])
	default:
		fmt.Fprintf(os.Stderr, "unknown proposal command: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "Usage: agenterm proposal status <proposal_id>")
		return 2
	}
}

func runProposalStatus(id string) int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		return 2
	}

	client := relay.NewClient(cfg)

	proposal, err := client.GetProposal(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error getting proposal: %v\n", err)
		return 2
	}

	return outputProposal(proposal)
}

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

	matched := false
	exitCode := 0
	for _, t := range agent.Targets() {
		if target != "all" && target != t.Name {
			continue
		}
		matched = true
		if err := t.Install(binaryPath, ""); err != nil {
			if errors.Is(err, hook.ErrAlreadyInstalled) {
				fmt.Fprintf(os.Stderr, "[%s] hook already installed\n", t.Name)
				continue
			}
			fmt.Fprintf(os.Stderr, "[%s] error: %v\n", t.Name, err)
			exitCode = 1
			continue
		}
		settingsPath, _ := t.SettingsPath()
		fmt.Fprintf(os.Stderr, "[%s] Installed %s hook in %s\n", t.Name, t.HookName, settingsPath)
	}

	if !matched {
		fmt.Fprintf(os.Stderr, "unknown target: %s (use claude, gemini, or omit for all)\n", target)
		return 2
	}
	return exitCode
}

func runHookUninstall(target string) int {
	matched := false
	exitCode := 0
	for _, t := range agent.Targets() {
		if target != "all" && target != t.Name {
			continue
		}
		matched = true
		if err := t.Uninstall(""); err != nil {
			if errors.Is(err, hook.ErrNotInstalled) {
				fmt.Fprintf(os.Stderr, "[%s] hook not found\n", t.Name)
				continue
			}
			fmt.Fprintf(os.Stderr, "[%s] error: %v\n", t.Name, err)
			exitCode = 1
			continue
		}
		settingsPath, _ := t.SettingsPath()
		fmt.Fprintf(os.Stderr, "[%s] Uninstalled hook from %s\n", t.Name, settingsPath)
	}

	if !matched {
		fmt.Fprintf(os.Stderr, "unknown target: %s (use claude, gemini, or omit for all)\n", target)
		return 2
	}
	return exitCode
}

func runGate(args []string) int {
	fs := flag.NewFlagSet("gate", flag.ContinueOnError)
	timeout := fs.Int("timeout", 60, "approval timeout in seconds")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
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

	if hookInput := gate.ParseHookInput(data); hookInput != nil {
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
func runGateHook(hookInput *gate.HookInput, timeout int) int {
	if hookInput.HookEventName == "PermissionRequest" {
		return runGatePermissionRequest(hookInput, timeout)
	}
	return runGateToolCheck(hookInput, timeout, agent.OutputterForEvent(hookInput.HookEventName))
}

// runGateToolCheck is the shared gate logic for rule-matched hooks (PreToolUse, BeforeTool).
func runGateToolCheck(hookInput *gate.HookInput, timeout int, out agent.Outputter) int {
	input := gate.ExtractCheckInput(hookInput)
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
	if errors.Is(err, relay.ErrGateDisabled) {
		fmt.Fprintf(os.Stderr, "gate: takeover not enabled, falling back to local prompt\n")
		if askLocallyInTerminal(input) {
			outputJSON(out.Allow("approved locally via AgenTerm"))
		} else {
			outputJSON(out.Deny("denied locally via AgenTerm"))
		}
		return 0
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "gate: approval error: %v\n", err)
		return 2
	}

	switch result.Decision {
	case "approved", "allow":
		outputJSON(out.Allow("approved via AgenTerm"))
	default:
		outputJSON(out.Deny(fmt.Sprintf("denied via AgenTerm: %s", rule.Description)))
	}
	return 0
}

// runGatePermissionRequest handles PermissionRequest hooks.
// Claude Code has already determined this action needs permission — no rule matching needed.
func runGatePermissionRequest(hookInput *gate.HookInput, timeout int) int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gate: config not found, blocking permission request\n")
		return 2
	}

	// Build a meaningful title and body for the proposal.
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
	out := agent.ClaudePermissionOutputter{}

	result, err := gate.RunPermissionGate(client, title, body, time.Duration(timeout)*time.Second)
	if errors.Is(err, relay.ErrGateDisabled) {
		fmt.Fprintf(os.Stderr, "gate: takeover not enabled, falling back to local prompt\n")
		if askLocallyInTerminal(fmt.Sprintf("Tool: %s\nInput: %s", title, body)) {
			outputJSON(out.Allow("approved locally via AgenTerm"))
		} else {
			outputJSON(out.Deny("denied locally via AgenTerm"))
		}
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
		outputJSON(out.Deny("denied via AgenTerm"))
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

func outputJSON(v interface{}) {
	json.NewEncoder(os.Stdout).Encode(v)
}

// outputProposal prints the proposal as JSON to stdout and returns the exit code.
func outputProposal(p *relay.Proposal) int {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(p); err != nil {
		fmt.Fprintf(os.Stderr, "error encoding output: %v\n", err)
		return 2
	}

	switch p.Status {
	case "approved", "remembered", "pending":
		return 0
	case "denied", "dismissed":
		return 1
	default:
		return 2
	}
}

// askLocallyInTerminal opens the user's terminal directly (/dev/tty) to prompt for approval
// bypassing stdin/stdout which are used for Hook JSON communication.
func askLocallyInTerminal(cmd string) bool {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot open terminal for local prompt, defaulting to deny: %v\n", err)
		return false
	}
	defer tty.Close()

	fmt.Fprintf(tty, "\n🛡️  [AgenTerm Local Security Approval]\n")
	fmt.Fprintf(tty, "Agent is requesting to execute the following command/tool:\n> %s\n\n", cmd)
	fmt.Fprintf(tty, "Allow execution? [y/N]: ")

	reader := bufio.NewReader(tty)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	return answer == "y" || answer == "yes"
}
