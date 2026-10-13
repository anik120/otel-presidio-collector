package presidioprocessor

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
)

// Config holds the settings for the redaction processor. Redaction is delegated
// to a data-anonymizer HTTP service; the redaction policy (entities, allow/deny
// lists, spaCy model) lives in that service, not in the collector.
type Config struct {
	// Endpoint is the base URL of the data-anonymizer service (it exposes POST /anonymize).
	Endpoint string `mapstructure:"endpoint"`
	// Attributes is the list of span attribute keys whose string values are redacted.
	Attributes []string `mapstructure:"attributes"`
	// Timeout bounds each HTTP call to the service.
	Timeout time.Duration `mapstructure:"timeout"`
}

func createDefaultConfig() component.Config {
	return &Config{
		Endpoint:   "http://localhost:5000",
		Attributes: []string{"request.input", "response.output", "rag.input"},
		Timeout:    10 * time.Second,
	}
}

// Validate checks that the service endpoint is set.
func (c *Config) Validate() error {
	if c.Endpoint == "" {
		return errors.New("endpoint must be set")
	}
	return nil
}
