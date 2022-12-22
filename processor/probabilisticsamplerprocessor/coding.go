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

type hexDigits int

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

	// SignMask selects the sign bit of an IEEE 754 floating point
	// number.
	SignMask = (1 << (SignificandWidth + ExponentWidth))

	// MinNormalExponent is the minimum exponent of a normalized
	// floating point: -1022.
	MinNormalExponent int32 = -ExponentBias + 1

	// MaxNormalExponent is the maximum exponent of a normalized
	// floating point: 1023.
	MaxNormalExponent int32 = ExponentBias

	// MinValue is the smallest normal number.
	MinValue = 0x1p-1022

	// MaxValue is the largest normal number.
	MaxValue = math.MaxFloat64
)

// GetNormalBase2 extracts the normalized base-2 fractional exponent.
// Unlike Frexp(), this returns k for the equation f x 2**k where f is
// in the range [1, 2).  Note that this function is not called for
// subnormal numbers.
func GetNormalBase2(value float64) int32 {
	rawBits := math.Float64bits(value)
	rawExponent := (int64(rawBits) & ExponentMask) >> SignificandWidth
	return int32(rawExponent - ExponentBias)
}

// GetSignificand returns the 52 bit (unsigned) significand as a
// signed value.
func GetSignificand(value float64) int64 {
	return int64(math.Float64bits(value)) & SignificandMask
}

func nf() *big.Float {
	return (&big.Float{}).SetMode(big.ToZero).SetPrec(96)
}

func zeros(n int) string {
	return strings.Repeat("0", n)
}

var one = big.NewFloat(1).SetPrec(96)

type sampleProb struct {
	Probability *big.Float

	LowCoded     string
	LowThreshold [7]byte
	LowProb      float64

	HighCoded     string
	HighThreshold [7]byte
	HighProb      float64
}

func codedToThreshold(s string) [7]byte {
	sig, shifts, _ := strings.Cut(s, "-")
	shift, _ := strconv.Atoi(shifts)
	var thresh [7]byte
	hex.Decode(thresh[:], []byte(
		zeros(shift)+sig+zeros(14-shift-len(sig))),
	)
	return thresh
}

func thresholdToProb(t [7]byte) float64 {
	// Make 56 bits into 52 bits
	var eight [8]byte
	copy(eight[1:8], t[:])
	b52 := binary.BigEndian.Uint64(eight[:]) >> 4
	// Let exp=0, make a floating point, then subtract 1.
	b52 |= ExponentBias << SignificandWidth
	return math.Float64frombits(b52) - 1

}

func formatCoded(bits int64, offset int) string {
	for bits&0xf == 0 {
		bits >>= 4
	}
	return fmt.Sprintf("%x-%d", bits, offset)
}

func (digits hexDigits) calc(ps string) sampleProb {
	// The probability is input as a string, can be parsed as a
	// rational number or a floating point.

	probAsFloat := nf()

	if strings.Count(ps, "/") == 1 {
		num, den, _ := strings.Cut(ps, "/")
		n, err1 := strconv.ParseInt(num, 0, 64)
		d, err2 := strconv.ParseInt(den, 0, 64)
		if err1 != nil || err2 != nil {
			panic(fmt.Errorf("%w: %s", err1, err2))
		}

		probAsFloat.SetRat(big.NewRat(n, d))
	} else {
		probAsFloat.Parse(ps, 0)
	}

	twoTo56 := nf().SetMantExp(one, 56)

	// subtracting 1, thresholdBits is now a 56 bit unsigned number.
	threshold := nf().Sub(nf().Mul(probAsFloat, twoTo56), one)

	thresholdBits, _ := threshold.Int64()

	if thresholdBits < 0 {
		// Probability underflow for values close to 2**-56.
		panic("underflow!")
	}

	lz := bits.LeadingZeros64(uint64(thresholdBits))
	nzbits := 64 - lz
	fullQuads := nzbits / 4
	shiftBy := 4 * (fullQuads - int(digits))
	if nzbits%4 != 0 {
		shiftBy += 4
	}

	zcnt := (56 - nzbits) / 4

	thresholdShifted := thresholdBits >> shiftBy

	result := sampleProb{
		Probability: probAsFloat,
	}

	var ideal [7]byte
	hex.Decode(ideal[:], []byte(fmt.Sprintf("%014x", thresholdBits)))

	// Note: we can remove trailing 0s from the thresholdShifted
	// variable.
	coded := formatCoded(thresholdShifted, zcnt)

	result.LowCoded = coded
	result.LowThreshold = codedToThreshold(coded)
	result.LowProb = thresholdToProb(result.LowThreshold)

	{
		incr := thresholdShifted + 1
		izcnt := zcnt
		if bits.LeadingZeros64(uint64(incr)) > bits.LeadingZeros64(uint64(thresholdShifted)) {
			incr >>= 4
			izcnt -= 1
		}
		nextCoded := formatCoded(incr, izcnt)

		result.HighCoded = nextCoded
		result.HighThreshold = codedToThreshold(nextCoded)
		result.HighProb = thresholdToProb(result.HighThreshold)
	}

	return result
}
