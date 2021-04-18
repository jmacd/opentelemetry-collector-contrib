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
	// memoryMetricsExporter is an Exporter that places metric
	// events into a short-term buffer for the purposes of:
	//
	// - temporal alignment
	// - re-aggregation
	// - de-duplication
	// - overlap resolution
	// - join support (e.g., for "up")
	memoryMetricsExporter struct {
		// config describes the window size, interval size,
		// and the ratio of past- and future-timestamped data
		// to maintain.
		config Config

		// oldestTime is te start timestamp of intervals[0]
		oldestTime time.Time

		// intervals is a circular buffer
		intervals []interval

		// ... more fields
	}

	// interval is one collection of streams, writers, and points.
	interval struct {
		streams map[streamKey]*stream
	}

	// streamKey identifies an "in-practice" stream.  unit and
	// kind are expected to be the same for a given name, but they
	// are treated as independent due to [want an good adjective
	// here] incompatibility.
	streamKey struct {
		// library is the instrumentation library
		library string

		// name is the metric name
		name string

		// unit are the units
		unit string

		// kind of the point kind, including monotonicity and
		// temporality, bucket style, etc.
		//
		// NOT included in kind: number type.
		kind kind
	}

	unixNanos uint64 // from the OTLP protocol
	kind      int    // an enum (possibly equal to the top-level `oneof`?)

	// stream is a map of writers.
	stream struct {
		writers map[coordinate]*writer
	}

	// coordinate describes a single writer
	coordinate struct {
		resource   attribute.Distinct
		attributes attribute.Distinct
	}

	// writer maintains a set of list of points.
	writer struct {
		points points
	}

	// point is a single point, described by
	point struct {
		startNanos unixNanos
		timeNanos  unixNanos
		external   attribute.Set

		// scalar case
		numberKind number.Kind
		scalar     number.Number

		// histogram case TODO
	}

	points []point
)

const (
	kindNone kind = iota
	kindSumMonotonicDelta
	kindSumMonotonicCumulative
	kindSumNonMonotonicDelta
	kindSumNonMonotonicCumulative
	kindHistogramExplicitDelta
	kindHistogramExplicitCumulative
	kindSummary
	kindGauge
)

func (e *memoryMetricsExporter) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		res := rm.Resource()
		resAttrs := res.Attributes()
		resKVs := make([]attribute.KeyValue, 0, resAttrs.Len())

		resAttrs.ForEach(func(resKey string, resVal pdata.AttributeValue) {
			var kv attribute.KeyValue
			switch resVal.Type() {
			case pdata.AttributeValueSTRING:
				kv = attribute.String(resKey, resVal.StringVal())
			case pdata.AttributeValueBOOL:
				kv = attribute.Bool(resKey, resVal.BoolVal())
			case pdata.AttributeValueINT:
				kv = attribute.Int64(resKey, resVal.IntVal())
			case pdata.AttributeValueDOUBLE:
				kv = attribute.Float64(resKey, resVal.DoubleVal())
			case pdata.AttributeValueMAP, pdata.AttributeValueARRAY,
				pdata.AttributeValueNULL:
				// TODO: error state, or format these; shrug don't do this.
			}
			resKVs = append(resKVs, kv)
		})

		resAttrSet := attribute.NewSet(resKVs...)

		ilms := rm.InstrumentationLibraryMetrics()

		for j := 0; j < ilms.Len(); j++ {
			ilm := ilms.At(j)
			library := ilm.InstrumentationLibrary().Name()

			ms := ilm.Metrics()
			for k := 0; k < ms.Len(); k++ {
				m := ms.At(k)
				var numberKind number.Kind

				switch m.DataType() {
				case pdata.MetricDataTypeIntGauge:
					numberKind = number.Int64Kind
					kind = kindGauge

					//dataPointCount += m.IntGauge().DataPoints().Len()
				case pdata.MetricDataTypeDoubleGauge:
					numberKind = number.Float64Kind
					kind = kindGauge

					//dataPointCount += m.DoubleGauge().DataPoints().Len()
				case pdata.MetricDataTypeIntSum:
					//dataPointCount += m.IntSum().DataPoints().Len()
				case pdata.MetricDataTypeDoubleSum:
					//dataPointCount += m.DoubleSum().DataPoints().Len()
				case pdata.MetricDataTypeIntHistogram:
					//dataPointCount += m.IntHistogram().DataPoints().Len()
				case pdata.MetricDataTypeHistogram:
					//dataPointCount += m.Histogram().DataPoints().Len()
				case pdata.MetricDataTypeSummary:
					//dataPointCount += m.Summary().DataPoints().Len()
				}
			}
		}
	}

	// interval := e.intervals[0]
	// stream, ok := interval.streams[...]

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
