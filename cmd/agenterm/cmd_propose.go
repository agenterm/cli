package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/agenterm/cli/internal/config"
	"github.com/agenterm/cli/internal/relay"
)

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

	// APN enforces a minimum expires_in of 60s; clamp to avoid silent mismatch.
	if *timeout < 60 {
		*timeout = 60
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

	proposal, err := client.CreateProposal("approval", *title, *body, relay.WithExpiresIn(*timeout+15))
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
