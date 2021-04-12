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
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/number"
)

// memoryMetricsExporter is an in-memory metrics buffer.
type (
	memoryMetricsExporter struct {
		config Config

		oldestTime time.Time
		intervals  []interval
	}

	mapkey struct {
		res   attribute.Distinct
		attrs attribute.Distinct
	}

	interval struct {
		m map[mapkey]*record
	}

	record struct {
		s []sample
	}

	sample struct {
		startNanos uint64
		duration   time.Duration
		external   attribute.Distinct

		value interface{} // @@@ or ...?
	}

	scalar number.Number
)

func (e *memoryMetricsExporter) ConsumeMetrics(_ context.Context, md pdata.Metrics) error {
	// @@@
	// What's the relationship between this code and Delta->Cumulative
	// (Implement memory option like OTel-Go?)
	return nil
}

func (e *memoryMetricsExporter) Start(context.Context, component.Host) error {
	// Timer to ... update intervals.
	return nil
}

// Shutdown stops the exporter and is invoked during shutdown.
func (e *memoryMetricsExporter) Shutdown(context.Context) error {
	return nil
}
