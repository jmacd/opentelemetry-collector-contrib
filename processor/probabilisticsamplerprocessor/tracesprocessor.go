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

package probabilisticsamplerprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/probabilisticsamplerprocessor"

import (
	"bytes"
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.uber.org/zap"
)

type traceSamplerProcessor struct {
	scaledSamplingRate uint32
	logger             *zap.Logger
	prob               sampleProb
}

// newTracesProcessor returns a processor.TracesProcessor that will perform head sampling according to the given
// configuration.
func newTracesProcessor(ctx context.Context, set component.ProcessorCreateSettings, cfg *Config, nextConsumer consumer.Traces) (component.TracesProcessor, error) {
	sampCfg := thresholdWidth(cfg.Precision).calc(fmt.Sprintf("%x", cfg.Probability))

	fmt.Printf(`Sampling probability: %.10f
Exact probability: %.10f
Encoded probability: %s
Sampling threshold: %x
`, sampCfg.Probability, sampCfg.LowProb, sampCfg.LowCoded, sampCfg.LowThreshold)

	tsp := &traceSamplerProcessor{
		// Adjust sampling percentage on private so recalculations are avoided.
		scaledSamplingRate: 1,
		logger:             set.Logger,
		prob:               sampCfg,
	}

	return processorhelper.NewTracesProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		tsp.processTraces,
		processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}))
}

func (tsp *traceSamplerProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	td.ResourceSpans().RemoveIf(func(rs ptrace.ResourceSpans) bool {
		rs.ScopeSpans().RemoveIf(func(ils ptrace.ScopeSpans) bool {
			ils.Spans().RemoveIf(func(s ptrace.Span) bool {
				tid := s.TraceID()
				return bytes.Compare(tid[9:16], tsp.prob.LowThreshold[:]) > 0
			})
			for i := 0; i < ils.Spans().Len(); i++ {
				span := ils.Spans().At(i)
				span.TraceState().FromRaw("ot=t:" + tsp.prob.LowCoded)

				// HACK: show the tracestate b/c the loggingexporter doesn't.
				span.Attributes().PutStr("otel.tracestate", "t:"+tsp.prob.LowCoded)
			}

			// Filter out empty ScopeMetrics
			return ils.Spans().Len() == 0
		})
		// Filter out empty ResourceMetrics
		return rs.ScopeSpans().Len() == 0
	})
	if td.ResourceSpans().Len() == 0 {
		return td, processorhelper.ErrSkipProcessingData
	}
	return td, nil
}
