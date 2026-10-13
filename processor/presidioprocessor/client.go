package presidioprocessor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// presidioClient calls the Presidio analyzer and anonymizer REST services.
// Redaction is a two-step flow: analyze the text to locate PII, then anonymize
// those spans. This mirrors the open-source Presidio service API, which has no
// single combined endpoint.
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

// blockedEntity is the entity type assigned to deny-list matches.
const blockedEntity = "BLOCKED_TERM"

type analyzeRequest struct {
	Text             string            `json:"text"`
	Language         string            `json:"language"`
	Entities         []string          `json:"entities,omitempty"`
	ScoreThreshold   float64           `json:"score_threshold,omitempty"`
	AllowList        []string          `json:"allow_list,omitempty"`
	AdHocRecognizers []adHocRecognizer `json:"ad_hoc_recognizers,omitempty"`
}

type adHocRecognizer struct {
	Name              string   `json:"name"`
	SupportedLanguage string   `json:"supported_language"`
	DenyList          []string `json:"deny_list"`
	SupportedEntity   string   `json:"supported_entity"`
}

// analyzerResult matches one entry of the presidio-analyzer /analyze response
// and is passed straight back as an analyzer_results entry to /anonymize.
type analyzerResult struct {
	EntityType string  `json:"entity_type"`
	Start      int     `json:"start"`
	End        int     `json:"end"`
	Score      float64 `json:"score"`
}

type anonymizeRequest struct {
	Text            string                      `json:"text"`
	AnalyzerResults []analyzerResult            `json:"analyzer_results"`
	Anonymizers     map[string]anonymizerConfig `json:"anonymizers"`
}

type anonymizerConfig struct {
	Type     string `json:"type"`
	NewValue string `json:"new_value,omitempty"`
}

type anonymizeResponse struct {
	Text string `json:"text"`
}

// Redact returns text with detected PII replaced. If the analyzer finds nothing,
// the original text is returned without calling the anonymizer.
func (c *presidioClient) Redact(ctx context.Context, text string) (string, error) {
	results, err := c.analyze(ctx, text)
	if err != nil {
		return "", fmt.Errorf("analyze: %w", err)
	}
	if len(results) == 0 {
		return text, nil
	}
	redacted, err := c.anonymize(ctx, text, results)
	if err != nil {
		return "", fmt.Errorf("anonymize: %w", err)
	}
	return redacted, nil
}

func (c *presidioClient) analyze(ctx context.Context, text string) ([]analyzerResult, error) {
	req := analyzeRequest{
		Text:           text,
		Language:       c.cfg.Language,
		Entities:       c.cfg.Entities,
		ScoreThreshold: c.cfg.ScoreThreshold,
		AllowList:      c.cfg.AllowList,
	}
	if len(c.cfg.BlockList) > 0 {
		req.AdHocRecognizers = []adHocRecognizer{{
			Name:              "block_list",
			SupportedLanguage: c.cfg.Language,
			DenyList:          c.cfg.BlockList,
			SupportedEntity:   blockedEntity,
		}}
		// When entities are restricted, the deny-list's custom entity must be in
		// the list too, or the analyzer filters its matches out.
		if len(c.cfg.Entities) > 0 {
			req.Entities = append(append([]string{}, c.cfg.Entities...), blockedEntity)
		}
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.post(ctx, c.cfg.AnalyzerEndpoint+"/analyze", body)
	if err != nil {
		return nil, err
	}
	var results []analyzerResult
	if err := json.Unmarshal(resp, &results); err != nil {
		return nil, fmt.Errorf("decode analyze response: %w", err)
	}
	return results, nil
}

func (c *presidioClient) anonymize(ctx context.Context, text string, results []analyzerResult) (string, error) {
	body, err := json.Marshal(anonymizeRequest{
		Text:            text,
		AnalyzerResults: results,
		Anonymizers: map[string]anonymizerConfig{
			"DEFAULT": {Type: "replace", NewValue: c.cfg.Replacement},
		},
	})
	if err != nil {
		return "", err
	}
	resp, err := c.post(ctx, c.cfg.AnonymizerEndpoint+"/anonymize", body)
	if err != nil {
		return "", err
	}
	var out anonymizeResponse
	if err := json.Unmarshal(resp, &out); err != nil {
		return "", fmt.Errorf("decode anonymize response: %w", err)
	}
	return out.Text, nil
}

func (c *presidioClient) post(ctx context.Context, url string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %d: %s", url, resp.StatusCode, string(data))
	}
	return data, nil
}
