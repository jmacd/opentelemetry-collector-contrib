module github.com/open-telemetry/opentelemetry-collector-contrib/extension/observer/k8sobserver

go 1.14

require (
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/observer v0.0.0-00010101000000-000000000000
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/k8sconfig v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/collector v0.18.0
	go.uber.org/zap v1.16.0
	k8s.io/api v0.20.1
	k8s.io/apimachinery v0.20.5
	k8s.io/client-go v0.20.1
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
)

replace github.com/open-telemetry/opentelemetry-collector-contrib/extension/observer => ../

replace github.com/open-telemetry/opentelemetry-collector-contrib/internal/k8sconfig => ../../../internal/k8sconfig
