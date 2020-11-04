module github.com/open-telemetry/opentelemetry-collector-contrib/exporter/sumologicexporter

go 1.14

require (
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/collector v0.13.1-0.20201020175630-99cb5b244aad
	go.uber.org/zap v1.16.0
)

replace go.opentelemetry.io/collector => github.com/SumoLogic/opentelemetry-collector v0.2.7-0.20201104121817-60b932c8bd9c
