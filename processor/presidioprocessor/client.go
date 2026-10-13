package presidioprocessor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// presidioClient calls the data-anonymizer service, which wraps the Presidio SDK
// (analyze + anonymize) behind one endpoint and returns text with PII replaced.
// One round-trip per attribute.
type presidioClient struct {
	cfg  *Config
	http *http.Client
}

func newPresidioClient(cfg *Config) *presidioClient {
	return &presidioClient{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

type anonymizeRequest struct {
	Text string `json:"text"`
}

type anonymizeResponse struct {
	Text string `json:"text"`
}

// Redact returns text with PII replaced by the data-anonymizer service.
func (c *presidioClient) Redact(ctx context.Context, text string) (string, error) {
	body, err := json.Marshal(anonymizeRequest{Text: text})
	if err != nil {
		return "", err
	}

	url := c.cfg.Endpoint + "/anonymize"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s returned %d: %s", url, resp.StatusCode, string(data))
	}

	var out anonymizeResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("decode anonymize response: %w", err)
	}
	return out.Text, nil
}
