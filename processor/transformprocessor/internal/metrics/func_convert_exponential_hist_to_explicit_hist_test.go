// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlmetric"
)

func TestConvertExponentialHistToExplicitHist(t *testing.T) {
	metric := pmetric.NewMetric()
	metric.SetName("test")
	metric.SetDescription("description")
	metric.SetUnit("ms")
	exponential := metric.SetEmptyExponentialHistogram()
	exponential.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	source := exponential.DataPoints().AppendEmpty()
	source.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Unix(100, 0)))
	source.SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(200, 0)))
	source.SetCount(16)
	source.SetScale(0)
	source.SetZeroThreshold(1)
	source.SetZeroCount(8)
	source.SetMin(-2)
	source.SetFlags(pmetric.DefaultDataPointFlags.WithNoRecordedValue(true))
	source.Attributes().PutStr("key", "value")
	source.Negative().BucketCounts().FromRaw([]uint64{4})
	source.Positive().BucketCounts().FromRaw([]uint64{4})
	exemplar := source.Exemplars().AppendEmpty()
	exemplar.SetDoubleValue(1.5)

	runConversion(t, metric, []float64{-2, -1, 0, 1, 2})

	require.Equal(t, pmetric.MetricTypeHistogram, metric.Type())
	assert.Equal(t, "test", metric.Name())
	assert.Equal(t, "description", metric.Description())
	assert.Equal(t, "ms", metric.Unit())
	assert.Equal(t, pmetric.AggregationTemporalityCumulative, metric.Histogram().AggregationTemporality())
	require.Equal(t, 1, metric.Histogram().DataPoints().Len())

	destination := metric.Histogram().DataPoints().At(0)
	assert.Equal(t, source.StartTimestamp(), destination.StartTimestamp())
	assert.Equal(t, source.Timestamp(), destination.Timestamp())
	assert.Equal(t, uint64(16), destination.Count())
	assert.Equal(t, []float64{-2, -1, 0, 1, 2}, destination.ExplicitBounds().AsRaw())
	assert.Equal(t, []uint64{0, 4, 4, 4, 4, 0}, destination.BucketCounts().AsRaw())
	assert.Equal(t, source.Flags(), destination.Flags())
	assert.False(t, destination.HasSum())
	assert.True(t, destination.HasMin())
	assert.Equal(t, -2.0, destination.Min())
	assert.False(t, destination.HasMax())
	value, ok := destination.Attributes().Get("key")
	require.True(t, ok)
	assert.Equal(t, "value", value.Str())
	require.Equal(t, 1, destination.Exemplars().Len())
	assert.Equal(t, 1.5, destination.Exemplars().At(0).DoubleValue())
}

func TestConvertExponentialHistToExplicitHistOverflow(t *testing.T) {
	metric := pmetric.NewMetric()
	source := metric.SetEmptyExponentialHistogram().DataPoints().AppendEmpty()
	source.SetCount(3)
	source.SetScale(0)
	source.Positive().SetOffset(10)
	source.Positive().BucketCounts().FromRaw([]uint64{3})

	runConversion(t, metric, []float64{1, 2, 3})

	destination := metric.Histogram().DataPoints().At(0)
	assert.Equal(t, []uint64{0, 0, 0, 3}, destination.BucketCounts().AsRaw())
	assert.Len(t, destination.BucketCounts().AsRaw(), destination.ExplicitBounds().Len()+1)
}

func TestConvertExponentialHistToExplicitHistNonExponential(t *testing.T) {
	metric := pmetric.NewMetric()
	metric.SetName("gauge")
	metric.SetEmptyGauge()
	expected := pmetric.NewMetric()
	metric.CopyTo(expected)

	runConversion(t, metric, []float64{1})

	assert.Equal(t, expected, metric)
}

func TestConvertExponentialHistToExplicitHistIsAtomicOnError(t *testing.T) {
	metric := pmetric.NewMetric()
	source := metric.SetEmptyExponentialHistogram().DataPoints().AppendEmpty()
	source.SetCount(2)
	source.SetScale(0)
	source.Positive().BucketCounts().FromRaw([]uint64{1})
	source.Attributes().PutStr("key", "value")
	expected := pmetric.NewMetric()
	metric.CopyTo(expected)

	exprFunc, err := convertExponentialHistToExplicitHist([]float64{1})
	require.NoError(t, err)
	ctx := ottlmetric.NewTransformContextPtr(pmetric.NewResourceMetrics(), pmetric.NewScopeMetrics(), metric)
	defer ctx.Close()
	_, err = exprFunc(t.Context(), ctx)

	require.ErrorContains(t, err, "source histogram count 2 does not match bucket count 1")
	assert.Equal(t, expected, metric)
}

func TestConvertExponentialHistToExplicitHistValidation(t *testing.T) {
	tests := []struct {
		name   string
		bounds []float64
		error  string
	}{
		{
			name:  "empty bounds",
			error: "explicit bounds cannot be empty",
		},
		{
			name:   "unordered bounds",
			bounds: []float64{2, 1},
			error:  "explicit bounds are not strictly increasing",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := convertExponentialHistToExplicitHist(test.bounds)
			require.ErrorContains(t, err, test.error)
		})
	}
}

func runConversion(t *testing.T, metric pmetric.Metric, bounds []float64) {
	t.Helper()
	exprFunc, err := convertExponentialHistToExplicitHist(bounds)
	require.NoError(t, err)
	ctx := ottlmetric.NewTransformContextPtr(pmetric.NewResourceMetrics(), pmetric.NewScopeMetrics(), metric)
	defer ctx.Close()
	_, err = exprFunc(t.Context(), ctx)
	require.NoError(t, err)
}
