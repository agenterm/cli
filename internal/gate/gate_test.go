package gate

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/agenterm/cli/internal/relay"
)

// mockRelayServer creates a test server that handles POST /proposals and GET /proposals/:id.
// finalStatus is the status returned on GET.
func mockRelayServer(finalStatus string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     "test-123",
				"status": "pending",
			})
		case "GET":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     "test-123",
				"status": finalStatus,
			})
		}
	}))
}

func newTestClient(serverURL string) *relay.Client {
	return &relay.Client{
		BaseURL:    serverURL,
		PushKey:    "test-key",
		HTTPClient: http.DefaultClient,
	}
}

func TestRunGate_NoMatch(t *testing.T) {
	// TC-G01: safe input, no match
	client := newTestClient("http://unused")
	rules := DefaultRules()

	result, err := RunGate(client, "ls", rules, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NeedsApproval {
		t.Fatal("expected NeedsApproval=false")
	}
	if result.Decision != "allow" {
		t.Fatalf("expected Decision=allow, got %q", result.Decision)
	}
}

func TestRunGate_Approved(t *testing.T) {
	// TC-G02: dangerous input + approved
	srv := mockRelayServer("approved")
	defer srv.Close()

	client := newTestClient(srv.URL)
	rules := DefaultRules()

	result, err := RunGate(client, "rm -rf /", rules, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.NeedsApproval {
		t.Fatal("expected NeedsApproval=true")
	}
	if result.Decision != "approved" {
		t.Fatalf("expected Decision=approved, got %q", result.Decision)
	}
}

func TestRunGate_Denied(t *testing.T) {
	// TC-G03: dangerous input + denied
	srv := mockRelayServer("denied")
	defer srv.Close()

	client := newTestClient(srv.URL)
	rules := DefaultRules()

	result, err := RunGate(client, "rm -rf /", rules, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decision != "denied" {
		t.Fatalf("expected Decision=denied, got %q", result.Decision)
	}
}

func TestRunGate_Expired(t *testing.T) {
	// TC-G04: dangerous input + expired → maps to denied
	srv := mockRelayServer("expired")
	defer srv.Close()

	client := newTestClient(srv.URL)
	rules := DefaultRules()

	result, err := RunGate(client, "rm -rf /", rules, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decision != "denied" {
		t.Fatalf("expected Decision=denied (expired→denied), got %q", result.Decision)
	}
}
