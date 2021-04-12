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
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

const (
	// The value of "type" key in configuration.
	typeStr = "memorymetrics"
)

// NewFactory creates a factory for the memory metrics exporter.
func NewFactory() component.ExporterFactory {
	return exporterhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		exporterhelper.WithMetrics(createMetricsExporter))
}

func createDefaultConfig() config.Exporter {
	return &Config{
		ExporterSettings: config.NewExporterSettings(typeStr),
		Window:           10 * time.Minute,
		Interval:         time.Minute,
	}
}

func createMetricsExporter(
	_ context.Context,
	_ component.ExporterCreateParams,
	cfg config.Exporter,
) (component.MetricsExporter, error) {
	config := *cfg.(*Config)

	num := config.Window / config.Interval
	// @@@ if cfg.Window % cfg.Interval != 0

	exporter := &memoryMetricsExporter{
		config:     config,
		oldestTime: time.Now(),
		intervals:  make([]interval, num),
	}

	return exporter, nil
}
