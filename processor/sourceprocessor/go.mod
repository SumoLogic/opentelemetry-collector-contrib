module github.com/open-telemetry/opentelemetry-collector-contrib/processor/sourceprocessor

go 1.14

require (
	github.com/stretchr/testify v1.6.1
	go.opencensus.io v0.22.4
	go.opentelemetry.io/collector v0.8.0
)

replace go.opentelemetry.io/collector => github.com/SumoLogic/opentelemetry-collector v0.8.1-0.20200816095155-88ae32e7a7f1
