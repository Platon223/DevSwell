package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	apiKey string
	from   string
}

func NewClient(apiKey, from string) *Client {
	return &Client{apiKey: apiKey, from: from}
}

// Send sends an HTML email via the Resend API.
func (c *Client) Send(ctx context.Context, to, subject, html string) error {
	payload := map[string]any{
		"from":    c.from,
		"to":      []string{to},
		"subject": subject,
		"html":    html,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encoding email payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend api returned %s: %s", resp.Status, respBody)
	}
	return nil
}
