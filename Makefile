OCB ?= ocb
BIN := ./_build/otelcol-presidio

.PHONY: help tidy build run deps-up deps-down clean

help: ## Show this help
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-12s %s\n", $$1, $$2}'

tidy: ## Resolve the processor module's dependencies
	cd processor/presidioprocessor && go mod tidy

build: tidy ## Build the custom collector via OCB
	$(OCB) --config builder-config.yaml

run: ## Run the collector against config.yaml
	$(BIN) --config config.yaml

deps-up: ## Build+start the data-anonymizer service and Jaeger
	podman compose -f docker-compose.yaml up -d --build

deps-down: ## Stop the data-anonymizer service and Jaeger
	podman compose -f docker-compose.yaml down

clean: ## Remove build output
	rm -rf _build
