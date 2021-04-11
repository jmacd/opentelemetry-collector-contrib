// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package memorymetricsexporter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
)

// memoryMetricsExporter is the implementation of file exporter that writes telemetry data to a file
// in Protobuf-JSON format.
type memoryMetricsExporter struct {
}

func (e *memoryMetricsExporter) ConsumeMetrics(_ context.Context, md pdata.Metrics) error {
	// @@@
	return nil
}

func (e *memoryMetricsExporter) Start(context.Context, component.Host) error {
	return nil
}

// Shutdown stops the exporter and is invoked during shutdown.
func (e *memoryMetricsExporter) Shutdown(context.Context) error {
	return nil
}
