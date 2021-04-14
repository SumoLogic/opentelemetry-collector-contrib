install:
	go install github.com/open-telemetry/opentelemetry-collector-builder@v0.24.0

build: install
	opentelemetry-collector-builder --config .otelcol-builder.yaml
