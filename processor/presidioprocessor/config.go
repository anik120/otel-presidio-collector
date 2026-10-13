package presidioprocessor

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
)

// Config holds the settings for the Presidio redaction processor.
type Config struct {
	// AnalyzerEndpoint is the base URL of the presidio-analyzer service.
	AnalyzerEndpoint string `mapstructure:"analyzer_endpoint"`
	// AnonymizerEndpoint is the base URL of the presidio-anonymizer service.
	AnonymizerEndpoint string `mapstructure:"anonymizer_endpoint"`
	// Attributes is the list of span attribute keys whose string values are redacted.
	Attributes []string `mapstructure:"attributes"`
	// Language is the Presidio analyzer language code (e.g. "en").
	Language string `mapstructure:"language"`
	// Replacement is the string substituted for detected PII.
	Replacement string `mapstructure:"replacement"`
	// Timeout bounds each HTTP call to Presidio.
	Timeout time.Duration `mapstructure:"timeout"`
}

func createDefaultConfig() component.Config {
	return &Config{
		AnalyzerEndpoint:   "http://localhost:5002",
		AnonymizerEndpoint: "http://localhost:5001",
		Attributes:         []string{"request.input", "response.output", "rag.input"},
		Language:           "en",
		Replacement:        "<REDACTED>",
		Timeout:            5 * time.Second,
	}
}

// Validate checks that the required endpoints are set.
func (c *Config) Validate() error {
	if c.AnalyzerEndpoint == "" {
		return errors.New("analyzer_endpoint must be set")
	}
	if c.AnonymizerEndpoint == "" {
		return errors.New("anonymizer_endpoint must be set")
	}
	return nil
}
