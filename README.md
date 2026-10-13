# otel-presidio-collector (PoC)

A custom OpenTelemetry Collector that runs a **Presidio PII-redaction processor**
in the pipeline. It validates whether Presidio (NER-grade redaction: names,
orgs, locations) can run as a collector processor, and measures the cost.

Module: `github.com/anik120/otel-presidio-collector` · collector version `0.0.1`
(built on upstream OpenTelemetry `0.162.0`).

## Why this exists

The supported RHOSDT collector (`rhosdt-3.11.1`) only ships regex-grade
redaction (`redaction`/`transform`) and nothing in it can call an external
service. Presidio is Python and the collector is Go, so "Presidio as a
processor" means a **custom Go processor that calls Presidio in microservice
mode** over HTTP, built into a derived collector image. This repo is that
processor plus the smallest collector that can run it.

See the design doc: `lightspeed-stack/docs/design/otel-collector-pii-redaction/design.md`.

## Layout

```
builder-config.yaml                     OCB manifest (dist.version 0.0.1)
config.yaml                             runtime collector config
docker-compose.yaml                     Presidio analyzer+anonymizer + Jaeger
processor/presidioprocessor/            the custom Go processor
  factory.go  config.go  processor.go  client.go
```

## Run the PoC

Prereqs: Go, `ocb`, podman.

```bash
make deps-up          # Presidio (5002 analyzer, 5001 anonymizer) + Jaeger (UI 16686)
make build            # OCB builds ./_build/otelcol-presidio
make run              # collector listens OTLP on :4317, forwards to Jaeger :4319
```

Then point LCORE at the collector and send a query with PII (reuses the existing
LCORE OTel demo):

```bash
export OTEL_SDK_DISABLED=false
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
export OTEL_EXPORTER_OTLP_PROTOCOL=grpc
# ... start lightspeed-stack ...

curl -s -X POST http://localhost:8080/v1/query \
  -H "Content-Type: application/json" \
  -d '{"query": "Anik from Red Hat (anbhatta@redhat.com) asks: what is Kubernetes?"}' | jq .
```

In Jaeger (http://localhost:16686), open the `query.handle_request` span. The
`request.input` tag should show the **name, org, and email redacted** — the NER
cases regex alone misses.

## What to measure (the point of the PoC)

- **Per-span latency** added by the analyze+anonymize round-trips.
- **Memory / throughput under load** (the `memory_limiter` is wired in on
  purpose) — a per-span HTTP call in the hot pipeline is where this strains.
- **Operational cost**: a derived image off the supported line, plus a Presidio
  sidecar and its spaCy model (~500MB `lg` / ~40MB `md` / ~12MB `sm`).

## Notes

- Presidio recently moved orgs (`data-privacy-stack/presidio`); the
  `mcr.microsoft.com/presidio-*` images in `docker-compose.yaml` may need
  updating to the new registry.
- This is a minimal distribution. To get the production-shaped artifact, port
  `processor/presidioprocessor` into a fork of
  `github.com/openshift/redhat-opentelemetry-collector`'s `manifest.yaml`.
