// Copyright Lightstep
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
package probabilisticsamplerprocessor

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"math/bits"
	"strconv"
	"strings"
)

// thresholdWidth is a type used to set the sampling threshold precision.
type thresholdWidth int

// Constants are copied out of go-expohisto internals, for working
// with IEEE double-width floating points.
const (
	// SignificandWidth is the size of an IEEE 754 double-precision
	// floating-point significand.
	SignificandWidth = 52
	// ExponentWidth is the size of an IEEE 754 double-precision
	// floating-point exponent.
	ExponentWidth = 11

	// SignificandMask is the mask for the significand of an IEEE 754
	// double-precision floating-point value: 0xFFFFFFFFFFFFF.
	SignificandMask = 1<<SignificandWidth - 1

	// ExponentBias is the exponent bias specified for encoding
	// the IEEE 754 double-precision floating point exponent: 1023.
	ExponentBias = 1<<(ExponentWidth-1) - 1

	// ExponentMask are set to 1 for the bits of an IEEE 754
	// floating point exponent: 0x7FF0000000000000.
	ExponentMask = ((1 << ExponentWidth) - 1) << SignificandWidth
)

// nf returns a new big float with 96
func nf() *big.Float {
	return (&big.Float{}).SetMode(big.ToZero).SetPrec(96)
}

// zeros returns n repeated "0"s.
func zeros(n int) string {
	return strings.Repeat("0", n)
}

// one is a big.Float unit
var one = big.NewFloat(1).SetPrec(96)

// sampleProb is the return result from thresholdWidth.calc(),
// contains all the useful information.
type sampleProb struct {
	// Probability is exactly as parsed with 96 bits of precision.
	Probability *big.Float

	// LowCoded is the smaller-probability threshold.
	LowCoded     string
	LowThreshold [7]byte
	LowProb      float64

	// TODO: an implementation could interpolate between the
	// larger-probability threshold to gain more accurate
	// span-to-metrics counting (especially when converting
	// adjusted counts to integers) however implementations will
	// note that traces on the boundary are probably incomplete,.
	//
	// This technique is used in
	// go-contrib/samplers/probability/consistent to select
	// between adjacent powers of two, by calculating the
	// likelihood of choosing LowCoded vs HighCoded to yield the
	// actual desired sampling probability.
	HighCoded     string
	HighThreshold [7]byte
	HighProb      float64
}

// codedToThreshold returns the 7-byte value as described by this
// encoding which is HHH-D containing H-many (configurable) hex digits
// and a leading-0s padding width in decimal.  The second part is
// always decimal. E.g.,
//
// Value    Threshold
// -----------------------
// "ef-2"   00ef0000000000
// "55-1"   05500000000000
// "555-10" 00000000005550
func codedToThreshold(s string) [7]byte {
	sig, shifts, _ := strings.Cut(s, "-")
	shift, _ := strconv.Atoi(shifts)
	var thresh [7]byte
	hex.Decode(thresh[:], []byte(
		zeros(shift)+sig+zeros(14-shift-len(sig))),
	)
	return thresh
}

// thresholdToProb calculates the probability as a float64 from the threshold.
func thresholdToProb(t [7]byte) float64 {
	// Make 56 bits into 52 bits.  TODO this skips corner cases near 0.
	var eight [8]byte
	copy(eight[1:8], t[:])
	b52 := binary.BigEndian.Uint64(eight[:]) >> 4
	// Produce a float64 with exponent 0 (bits ExponentBias) gives
	// the threshold as a fraction in the range [1, 2).  Make a
	// floating point, then subtract 1 for the range [0, 1).
	b52 |= ExponentBias << SignificandWidth
	return math.Float64frombits(b52) - 1

}

// formatCoded takes the completely-significant bits of a threshold
// and the zero-padding offset, returns.
func formatCoded(bits int64, offset int) string {
	// Note: discard trailing zeros, not interesting.
	for bits&0xf == 0 {
		bits >>= 4
	}
	return fmt.Sprintf("%x-%d", bits, offset)
}

// calc returns the calculated low and high probability thresholds.
func (thresholdWidth thresholdWidth) calc(ps string) sampleProb {
	// The probability is input as a string, can be parsed as a
	// rational number or a floating point.

	probAsFloat := nf()

	if strings.Count(ps, "/") == 1 {
		// Parse a rational number.
		num, den, _ := strings.Cut(ps, "/")
		n, err1 := strconv.ParseInt(num, 0, 64)
		d, err2 := strconv.ParseInt(den, 0, 64)
		if err1 != nil || err2 != nil {
			panic(fmt.Errorf("%w: %s", err1, err2))
		}

		probAsFloat.SetRat(big.NewRat(n, d))
	} else {
		// Parse a floating point number (any supported base).
		probAsFloat.Parse(ps, 0)
	}

	// Build a number 2**56, which is 1 greater than the 56-bit
	// maximum trace-random field expressed in bits.
	twoTo56 := nf().SetMantExp(one, 56)

	// Multiply the probability times 2**56.
	product := nf().Mul(probAsFloat, twoTo56)

	// Subtracting 1, thresholdBits is now a 56 bit unsigned number.
	// equal to the limiting trace-random threshold.  Trace-random values
	// less than or equal to the threshold should be sampled.
	threshold := nf().Sub(product, one)

	// Return to a machine word with up to 56 bits.
	thresholdBits, _ := threshold.Int64()

	// TODO: test the boundary conditions!
	if thresholdBits < 0 {
		// Probability underflow for values close to 2**-56.
		panic("underflow!")
	}

	// Below, calculate how many leading hex-zeros can be stripped.

	// sigBits is the number of significant bits, considering
	// leading zeros in the 64-bit wide representation.
	sigBits := 64 - bits.LeadingZeros64(uint64(thresholdBits))

	// fullQuads is Floor(sigBits/4)
	fullQuads := sigBits / 4

	// shiftBy is the difference between fullQuads and the
	// threshold width, multiplied by 4 (i.e., number of bits in a
	// hex digit).
	shiftBy := 4 * (fullQuads - int(thresholdWidth))
	if sigBits%4 != 0 {
		// When Floor(sigBits/4) rounds down, we have to shift
		// an extra hex width for the partial hex digit.
		shiftBy += 4
	}

	// zcnt is how many leading bits were stripped.  Note there is
	// a redundent calculation here b/c
	digitsStripped := (56 - sigBits) / 4

	// thresholdShifted is the final threshold value.
	thresholdShifted := thresholdBits >> shiftBy

	// result contains all the useful values.
	result := sampleProb{
		Probability: probAsFloat,
	}

	// calculate the coded probability, the corresponding
	// threshold, and the exact corresponding float64.
	coded := formatCoded(thresholdShifted, digitsStripped)
	result.LowCoded = coded
	result.LowThreshold = codedToThreshold(coded)
	result.LowProb = thresholdToProb(result.LowThreshold)

	// Calculate the next threshold.
	thresholdPlus1 := thresholdShifted + 1
	idigitsStripped := digitsStripped
	// When the threshold overflows, we have to shift again.
	// TODO haha this needs a test or two!
	if bits.LeadingZeros64(uint64(thresholdPlus1)) > bits.LeadingZeros64(uint64(thresholdShifted)) {
		thresholdPlus1 >>= 4
		idigitsStripped -= 1
	}

	nextCoded := formatCoded(thresholdPlus1, idigitsStripped)
	result.HighCoded = nextCoded
	result.HighThreshold = codedToThreshold(nextCoded)
	result.HighProb = thresholdToProb(result.HighThreshold)
	return result
}
