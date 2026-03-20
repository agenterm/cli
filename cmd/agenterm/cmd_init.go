package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/agenterm/cli/internal/config"
	"github.com/agenterm/cli/internal/relay"
)

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
	client := relay.NewClient(cfg)
	if err := client.Ping(); err != nil {
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
