package relay

import (
	"fmt"
	"net/http"
	"strings"
)

// ValidatePushKey checks that a push key is accepted by the relay server.
// It sends a test POST /proposals request and treats HTTP 401 as invalid.
func ValidatePushKey(relayURL, pushKey string) error {
	url := strings.TrimRight(relayURL, "/") + "/proposals"
	body := `{"type":"status","title":"__validate__"}`
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+pushKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return fmt.Errorf("connecting to relay: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid push key (HTTP 401)")
	}
	return nil
}
