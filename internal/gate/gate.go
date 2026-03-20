package gate

import (
	"fmt"
	"time"

	"github.com/agenterm/cli/internal/relay"
)

// expiresInBuffer is extra seconds added to server-side proposal expiry
// so it outlives the client-side wait, avoiding a race where the CLI times out
// but the proposal is still pending on the server.
const expiresInBuffer = 15

// GateResult holds the outcome of a gate check.
type GateResult struct {
	NeedsApproval bool   `json:"needs_approval"`
	Rule          string `json:"rule,omitempty"`
	Decision      string `json:"decision"`
}

// ProposalService abstracts the relay operations needed for gate checks.
// This allows RunGate to be tested without a real HTTP client.
type ProposalService interface {
	CreateProposal(pType, title, body string, opts ...relay.CreateOption) (*relay.Proposal, error)
	WaitForProposal(id string, timeout time.Duration) (*relay.Proposal, error)
}

// RunGate checks input against rules and, if matched, creates an approval proposal and waits.
func RunGate(svc ProposalService, input string, rules []Rule, timeout time.Duration) (*GateResult, error) {
	matched, rule := MatchesAny(input, rules)
	if !matched {
		return &GateResult{NeedsApproval: false, Decision: "allow"}, nil
	}

	proposal, err := submitAndWait(svc, rule.Description, input, timeout)
	if err != nil {
		return nil, err
	}

	return &GateResult{
		NeedsApproval: true,
		Rule:          rule.Description,
		Decision:      normalizeStatus(proposal.Status),
	}, nil
}

// RunPermissionGate creates a direct approval proposal (no rule matching) and waits for a response.
// Used for events like PermissionRequest where the agent has already determined approval is needed.
func RunPermissionGate(svc ProposalService, title, body string, timeout time.Duration) (*GateResult, error) {
	proposal, err := submitAndWait(svc, title, body, timeout)
	if err != nil {
		return nil, err
	}

	return &GateResult{
		NeedsApproval: true,
		Decision:      normalizeStatus(proposal.Status),
	}, nil
}

// submitAndWait creates an approval proposal and waits for a terminal response.
func submitAndWait(svc ProposalService, title, body string, timeout time.Duration) (*relay.Proposal, error) {
	proposal, err := svc.CreateProposal("approval", title, body, relay.WithExpiresIn(int(timeout.Seconds())+expiresInBuffer))
	if err != nil {
		return nil, fmt.Errorf("creating approval proposal: %w", err)
	}
	proposal, err = svc.WaitForProposal(proposal.ID, timeout)
	if err != nil {
		return nil, fmt.Errorf("waiting for approval: %w", err)
	}
	return proposal, nil
}

func normalizeStatus(status string) string {
	switch status {
	case "remembered":
		return "approved"
	case "dismissed", "expired":
		return "denied"
	default:
		return status
	}
}
