module github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sprocessor

go 1.13

require (
	github.com/census-instrumentation/opencensus-proto v0.2.1
	github.com/open-telemetry/opentelemetry-collector v0.2.7
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/stretchr/testify v1.4.0
	go.opencensus.io v0.22.3
	go.uber.org/zap v1.13.0
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v12.0.0+incompatible
)
