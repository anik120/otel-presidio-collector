# otel-presidio-collector 

A custom OpenTelemetry Collector that runs a **Presidio PII-redaction processor**
in the pipeline. Uses https://gitlab.cee.redhat.com/uxe-data-ai-solutions/data-anonymizer
as the Presidio fork(with customization applied to adhere to Red Hat compliance policies). 

## Architecture

```
  lightspeed-stack (LCORE)
        │  OTLP  :4317
        ▼
  ┌──────────────────────────────────────────────────────┐
  │ otelcol-presidio   (this repo — custom OCB collector)  │
  │   pipeline:  otlp → memory_limiter → presidio → batch  │
  │                                 │                      │
  │                                 │  POST /anonymize     │
  │                                 ▼                      │
  │        data-anonymizer service        :5000           │
  │        FastAPI → data-anonymizer lib (Presidio SDK)    │
  │                                 │                      │
  │          redacted text ◀────────┘                      │
  │        presidio writes it back onto the span           │
  └─────────────────────────┬──────────────────────────────┘
                            │  OTLP  :4319
                            ▼
                      Jaeger   (UI :16686)
```

The pipeline, stage by stage (`config.yaml`):

- **otlp** — receives spans from LCORE over OTLP gRPC (`:4317`).
- **memory_limiter** — first on purpose: when the collector's memory climbs (soft
  384 MiB / hard 512 MiB here) it refuses incoming data and backpressures the
  sender, so a flood of traces waiting on slow anonymizer calls can't OOM it.
- **presidio** (this collector's custom processor) — for each configured attribute
  (`request.input`, `response.output`, `rag.input`) it POSTs the text to the
  data-anonymizer service and writes the redacted result back onto the span. One
  HTTP round-trip per attribute.
- **batch** — groups the redacted spans before export (fewer, larger OTLP writes
  to Jaeger). It runs *after* presidio, so it makes the export efficient but does
  not reduce the per-attribute anonymizer calls.

[data-anonymizer](https://gitlab.cee.redhat.com/uxe-data-ai-solutions/data-anonymizer) is Python and the collector is Go, so NER-grade
redaction in the pipeline means a **custom Go processor that calls the anonymizer
over HTTP**, built into a derived collector image. This repo is that processor,
the minimal collector that runs it, and the service wrapper.

## Run the collector locally

Prereqs: Go 1.25.6+, [`ocb`](https://opentelemetry.io/docs/collector/extend/ocb/#install-the-opentelemetry-collector-builder), podman.

```bash
make deps-up          # data-anonymizer service (:5000) + Jaeger (UI :16686)
make build            # OCB builds ./_build/otelcol-presidio
make run              # collector listens OTLP on :4317, forwards to Jaeger :4319
```

Then point LCORE at the collector and send a query with PII:

```bash
export OTEL_SDK_DISABLED=false
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
export OTEL_EXPORTER_OTLP_PROTOCOL=grpc
# ... start lightspeed-stack ...

curl -s -X POST http://localhost:8080/v1/query \
  -H "Content-Type: application/json" \
  -d '{"query": "Sam from Red Hat (sam@redhat.com) asks: what is Kubernetes?"}' | jq .
```

In Jaeger (http://localhost:16686), open the `query.handle_request` span. The
`request.input` tag should show the **org, email etc redacted** — the NER
cases regex alone misses.
