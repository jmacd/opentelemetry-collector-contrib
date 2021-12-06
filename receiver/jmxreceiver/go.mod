module github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jmxreceiver

go 1.14

require (
	github.com/shirou/gopsutil v3.21.11+incompatible
	github.com/stretchr/testify v1.6.1
	github.com/testcontainers/testcontainers-go v0.9.0
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.opentelemetry.io/collector v0.18.0
	go.uber.org/atomic v1.7.0
	go.uber.org/zap v1.16.0
)

replace github.com/docker/docker => github.com/docker/engine v17.12.0-ce-rc1.0.20200309214505-aa6a9891b09c+incompatible
