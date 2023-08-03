package sampling

import (
	"encoding/binary"
	"errors"
	"strconv"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

var (
	// ErrRValueSize is returned for r-values != NumHexDigits hex digits.
	ErrRValueSize = errors.New("r-value must have 14 hex digits")
)

const (
	// LeastHalfTraceIDThresholdMask is the mask to use on the
	// least-significant half of the TraceID, i.e., bytes 8-15.
	// Because this is a 56 bit mask, the result after masking is
	// the unsigned value of bytes 9 through 15.
	LeastHalfTraceIDThresholdMask = MaxAdjustedCount - 1
)

// Randomness may be derived from r-value or TraceID.
type Randomness struct {
	// randomness is in the range [0, MaxAdjustedCount-1]
	unsigned uint64
}

// Randomness is the value we compare with Threshold in ShouldSample.
func RandomnessFromTraceID(id pcommon.TraceID) Randomness {
	return Randomness{
		unsigned: binary.BigEndian.Uint64(id[8:]) & LeastHalfTraceIDThresholdMask,
	}
}

// Unsigned is an unsigned integer that scales with the randomness
// value.  This is useful to compare two randomness values without
// floating point conversions.
func (r Randomness) Unsigned() uint64 {
	return r.unsigned
}

// RValueToRandomness parses NumHexDigits hex bytes into a Randomness.
func RValueToRandomness(s string) (Randomness, error) {
	if len(s) != NumHexDigits {
		return Randomness{}, ErrRValueSize
	}

	unsigned, err := strconv.ParseUint(s, hexBase, 64)
	if err != nil {
		return Randomness{}, err
	}

	return Randomness{
		unsigned: unsigned,
	}, nil
}
