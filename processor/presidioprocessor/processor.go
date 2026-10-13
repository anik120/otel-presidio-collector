package presidioprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type presidioProcessor struct {
	cfg    *Config
	logger *zap.Logger
	client *presidioClient
	next   consumer.Traces
}

func newProcessor(cfg *Config, logger *zap.Logger, next consumer.Traces) *presidioProcessor {
	return &presidioProcessor{
		cfg:    cfg,
		logger: logger,
		client: newPresidioClient(cfg),
		next:   next,
	}
}

// Capabilities reports that this processor mutates the trace data in place.
func (p *presidioProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (p *presidioProcessor) Start(_ context.Context, _ component.Host) error { return nil }

func (p *presidioProcessor) Shutdown(_ context.Context) error { return nil }

func (p *presidioProcessor) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	resourceSpans := td.ResourceSpans()
	for i := 0; i < resourceSpans.Len(); i++ {
		scopeSpans := resourceSpans.At(i).ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			spans := scopeSpans.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				p.redactAttributes(ctx, spans.At(k).Attributes())
			}
		}
	}
	return p.next.ConsumeTraces(ctx, td)
}

// redactAttributes replaces PII in each configured string attribute. On any
// Presidio error it logs and leaves the attribute untouched, so a redaction
// failure never drops the span (fail-closed on content, fail-open on delivery
// is a deliberate PoC choice — revisit for production).
func (p *presidioProcessor) redactAttributes(ctx context.Context, attrs pcommon.Map) {
	for _, key := range p.cfg.Attributes {
		v, ok := attrs.Get(key)
		if !ok || v.Type() != pcommon.ValueTypeStr {
			continue
		}
		original := v.Str()
		if original == "" {
			continue
		}
		redacted, err := p.client.Redact(ctx, original)
		if err != nil {
			p.logger.Warn("presidio redaction failed; leaving attribute unchanged",
				zap.String("attribute", key), zap.Error(err))
			continue
		}
		attrs.PutStr(key, redacted)
	}
}
