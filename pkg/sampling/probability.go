// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sampling // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/sampling"

import (
	"errors"
	"math"
)

// ErrProbabilityRange is returned when a value should be in the range [1/MaxAdjustedCount, 1].
var ErrProbabilityRange = errors.New("sampling probability out of the range [1/MaxAdjustedCount, 1]")

// MinSamplingProbability is the smallest representable probability
// and is the inverse of MaxAdjustedCount.
const MinSamplingProbability = 1.0 / float64(MaxAdjustedCount)

// probabilityInRange tests MinSamplingProb <= prob <= 1.
func probabilityInRange(prob float64) bool {
	return prob >= MinSamplingProbability && prob <= 1
}

// ProbabilityToThreshold converts a probability to a Threshold.  It
// returns an error when the probability is out-of-range.
func ProbabilityToThreshold(prob float64) (Threshold, error) {
	return ProbabilityToThresholdWithPrecision(prob, NumHexDigits)
}

// ProbabilityToThresholdWithPrecision is like ProbabilityToThreshold
// with support for reduced precision.  The `precision` argument determines
// how many significant hex digits will be used to encode the exact
// probability.
func ProbabilityToThresholdWithPrecision(fraction float64, precision int) (Threshold, error) {
	// Assume full precision at 0.
	if precision == 0 {
		precision = NumHexDigits
	}
	if !probabilityInRange(fraction) {
		return AlwaysSampleThreshold, ErrProbabilityRange
	}
	// Special case for prob == 1.
	if fraction == 1 {
		return AlwaysSampleThreshold, nil
	}

	_, expF := math.Frexp(fraction)
	_, expR := math.Frexp(1 - fraction)
	precision = min(NumHexDigits, max(precision+expF/-hexBits, precision+expR/-hexBits))

	// Compute the threshold
	scaled := uint64(math.Round(fraction * float64(MaxAdjustedCount)))
	threshold := MaxAdjustedCount - scaled

	// Round to the specified precision, if less than the maximum.
	if shift := hexBits * (NumHexDigits - precision); shift != 0 {
		half := uint64(1) << (shift - 1)
		threshold += half
		threshold >>= shift
		threshold <<= shift
	}

	return Threshold{
		unsigned: threshold,
	}, nil
}

// Probability is the sampling ratio in the range [MinSamplingProb, 1].
func (t Threshold) Probability() float64 {
	return float64(MaxAdjustedCount-t.unsigned) / float64(MaxAdjustedCount)
}
