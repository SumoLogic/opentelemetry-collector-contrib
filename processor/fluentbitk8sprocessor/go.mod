module github.com/open-telemetry/opentelemetry-collector-contrib/processor/fluentbitk8sprocessor

go 1.14

replace go.opentelemetry.io/collector => github.com/SumoLogic/opentelemetry-collector v0.8.0

require (
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/collector v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.15.0
)
