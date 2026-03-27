package relay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// HookResponse is the response from POST /hooks.
type HookResponse struct {
	Mode     string    `json:"mode"`               // "decision" or "observability"
	Proposal *Proposal `json:"proposal,omitempty"`  // present when mode is "decision"
}

// ForwardHook sends raw hook JSON to the relay and returns the response.
// For decision hooks the caller should follow up with WaitForProposal.
func (c *Client) ForwardHook(hookJSON []byte) (*HookResponse, error) {
	resp, err := c.doRequest("POST", "/hooks", bytes.NewReader(hookJSON))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}

	b, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("forward hook: HTTP %d: %s", resp.StatusCode, parseErrorBody(b))
	}

	var hookResp HookResponse
	if err := json.Unmarshal(b, &hookResp); err != nil {
		return nil, fmt.Errorf("decoding hook response: %w", err)
	}

	// Check for gate disabled response (proposal with status "disabled").
	if hookResp.Proposal != nil && hookResp.Proposal.Status == "disabled" {
		return &hookResp, ErrGateDisabled
	}

	return &hookResp, nil
}
