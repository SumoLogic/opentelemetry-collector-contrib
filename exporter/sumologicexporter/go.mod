module github.com/open-telemetry/opentelemetry-collector-contrib/exporter/sumologicexporter

go 1.14

replace go.opentelemetry.io/collector => github.com/SumoLogic/opentelemetry-collector v0.8.0

require (
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/stackdriverexporter v0.7.0 // indirect
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/collector v0.7.0
	go.uber.org/zap v1.15.0
)
