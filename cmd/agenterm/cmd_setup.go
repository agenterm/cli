package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/agenterm/cli/internal/agent"
	"github.com/agenterm/cli/internal/config"
	"github.com/agenterm/cli/internal/hook"
	"github.com/agenterm/cli/internal/relay"
)

func runSetup(args []string) int {
	fmt.Fprintln(os.Stderr, "AgenTerm Setup")
	fmt.Fprintln(os.Stderr, "==============")
	fmt.Fprintln(os.Stderr)

	// Step 1: Detect installed agents
	fmt.Fprintln(os.Stderr, "Detecting installed agents...")
	var detected []agent.HookTarget
	for _, t := range agent.Targets() {
		if settingsPath, err := t.SettingsPath(); err == nil {
			if _, err := os.Stat(settingsPath); err == nil {
				detected = append(detected, t)
				fmt.Fprintf(os.Stderr, "  [found] %s (%s)\n", t.Name, settingsPath)
			}
		}
	}

	if len(detected) == 0 {
		fmt.Fprintln(os.Stderr, "  No supported agents found.")
		fmt.Fprintln(os.Stderr, "  Supported: claude, gemini")
		fmt.Fprintln(os.Stderr, "  Install Claude Code or Gemini CLI, then re-run 'agenterm setup'.")
		return 1
	}
	fmt.Fprintln(os.Stderr)

	// Step 2: Configure relay connection
	cfg, _ := config.Load()
	if cfg.PushKey != "" {
		fmt.Fprintln(os.Stderr, "Existing configuration found.")
		fmt.Fprintf(os.Stderr, "  Relay: %s\n", cfg.RelayURL)
		fmt.Fprintf(os.Stderr, "  Push key: %s...%s\n", cfg.PushKey[:4], cfg.PushKey[len(cfg.PushKey)-4:])
		fmt.Fprintln(os.Stderr)
	} else {
		reader := bufio.NewReader(os.Stdin)

		fmt.Fprintf(os.Stderr, "Relay URL (default: %s): ", config.DefaultRelayURL)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line != "" {
			cfg.RelayURL = line
		}

		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "To get your push key:")
		fmt.Fprintln(os.Stderr, "  1. Open AgenTerm on your iPhone/iPad")
		fmt.Fprintln(os.Stderr, "  2. Go to Settings > Account")
		fmt.Fprintln(os.Stderr, "  3. Tap your account to see Account Details")
		fmt.Fprintln(os.Stderr, "  4. Copy the Push Key")
		fmt.Fprintln(os.Stderr)
		fmt.Fprint(os.Stderr, "Push Key: ")
		line, _ = reader.ReadString('\n')
		cfg.PushKey = strings.TrimSpace(line)

		if cfg.PushKey == "" {
			fmt.Fprintln(os.Stderr, "error: push key is required")
			return 1
		}
	}

	// Step 3: Validate push key
	fmt.Fprint(os.Stderr, "Validating push key... ")
	client := relay.NewClient(cfg)
	if err := client.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "failed\nerror: %v\n", err)
		if strings.Contains(err.Error(), "401") {
			fmt.Fprintln(os.Stderr, "The push key may have expired. Copy a fresh key from the AgenTerm app.")
		}
		return 1
	}
	fmt.Fprintln(os.Stderr, "ok")

	// Step 4: Save config
	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "error saving config: %v\n", err)
		return 1
	}
	p, _ := config.ConfigPath()
	fmt.Fprintf(os.Stderr, "Configuration saved to %s\n", p)
	fmt.Fprintln(os.Stderr)

	// Step 5: Install hooks for detected agents
	fmt.Fprintln(os.Stderr, "Installing hooks...")
	binaryPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot determine binary path: %v\n", err)
		return 1
	}

	allOk := true
	for _, t := range detected {
		if err := t.Install(binaryPath, ""); err != nil {
			if errors.Is(err, hook.ErrAlreadyInstalled) {
				fmt.Fprintf(os.Stderr, "  [%s] hook already installed\n", t.Name)
			} else {
				fmt.Fprintf(os.Stderr, "  [%s] error: %v\n", t.Name, err)
				allOk = false
			}
		} else {
			settingsPath, _ := t.SettingsPath()
			fmt.Fprintf(os.Stderr, "  [%s] hook installed (%s)\n", t.Name, settingsPath)
		}
	}

	fmt.Fprintln(os.Stderr)
	if allOk {
		fmt.Fprintln(os.Stderr, "Setup complete! AgenTerm will now send approval requests to your phone.")
		fmt.Fprintln(os.Stderr, "Try running Claude Code or Gemini — you'll get push notifications for tool approvals.")
	} else {
		fmt.Fprintln(os.Stderr, "Setup completed with errors. Check the messages above.")
	}
	return 0
}
