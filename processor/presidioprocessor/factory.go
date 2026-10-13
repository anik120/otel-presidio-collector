// Package presidioprocessor is an OpenTelemetry Collector processor that redacts
// PII from span attributes by calling a Presidio analyzer/anonymizer service.
package presidioprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
)

var componentType = component.MustNewType("presidio")

const stability = component.StabilityLevelDevelopment

// NewFactory returns a factory for the Presidio redaction processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		componentType,
		createDefaultConfig,
		processor.WithTraces(createTracesProcessor, stability),
	)
}

func createTracesProcessor(
	_ context.Context,
	set processor.Settings,
	cfg component.Config,
	next consumer.Traces,
) (processor.Traces, error) {
	return newProcessor(cfg.(*Config), set.Logger, next), nil
}
